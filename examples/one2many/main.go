package main

import (
	"fmt"
	"net/url"
	"sync"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	rtcrtmp "github.com/notedit/rtc-rtmp"
	"github.com/notedit/rtmp-lib"
	"github.com/notedit/rtmp-lib/av"
	"github.com/notedit/rtmp-lib/pubsub"
	"github.com/pion/webrtc/v2"
)

type Channel struct {
	que *pubsub.Queue
}

var channels = map[string]*Channel{}

var routers = map[string]*rtcrtmp.RTCRouter{}

func startRtmp() {

	l := &sync.RWMutex{}

	server := rtmp.NewServer(1024)
	server.Addr = ":1935"

	server.HandlePublish = func(conn *rtmp.Conn) {

		l.Lock()
		ch := channels[conn.URL.Path]

		if ch == nil {
			ch = &Channel{}
			ch.que = pubsub.NewQueue()
			ch.que.SetMaxGopCount(1)
			channels[conn.URL.Path] = ch
		}
		l.Unlock()

		fmt.Println("publish ", conn.URL.Path)

		var err error

		var streams []av.CodecData

		if streams, err = conn.Streams(); err != nil {
			fmt.Println(err)
			return
		}

		ch.que.WriteHeader(streams)

		for {
			packet, err := conn.ReadPacket()
			if err != nil {
				fmt.Println(err)
				break
			}

			ch.que.WritePacket(packet)
		}

		l.Lock()
		delete(channels, conn.URL.Path)
		l.Unlock()

		l.Lock()
		router := routers[conn.URL.Path]
		if router != nil {
			router.Stop()
		}
		delete(routers, conn.URL.Path)
		l.Unlock()

		ch.que.Close()
	}

	server.HandlePlay = func(conn *rtmp.Conn) {

		l.RLock()
		ch := channels[conn.URL.Path]
		l.RUnlock()

		fmt.Println("play  ", conn.URL.Path)

		if ch != nil {

			cursor := ch.que.Latest()

			streams, err := cursor.Streams()

			if err != nil {
				panic(err)
			}

			conn.WriteHeader(streams)

			for {
				packet, err := cursor.ReadPacket()
				if err != nil {
					break
				}
				conn.WritePacket(packet)
			}
		}
	}

	err := server.ListenAndServe()
	if err != nil {
		panic(err)
	}
}

func index(c *gin.Context) {

	fmt.Println("hello world")
	c.HTML(200, "index.html", gin.H{})
}

func pullstream(c *gin.Context) {

	var data struct {
		SDP       string `json:"sdp"`
		StreamURL string `json:"streamurl"`
	}

	if err := c.ShouldBind(&data); err != nil {
		fmt.Println(err)
		c.JSON(200, gin.H{
			"s": 10001,
			"e": err,
		})
		return
	}

	u, err := url.Parse(data.StreamURL)

	if err != nil {
		fmt.Println("error", err)
		c.JSON(200, gin.H{
			"s": 10001,
			"e": err,
		})
		return
	}

	pullURL := "rtmp://localhost/" + u.Path

	fmt.Println("pullURL ===", pullURL)

	router, err := rtcrtmp.NewRTCRouter(pullURL)

	if err != nil {
		fmt.Println("error", err)
		c.JSON(200, gin.H{
			"s": 10001,
			"e": err,
		})
		return
	}

	transport, err := router.CreateSubscriber()

	if err != nil {
		fmt.Println("error", err)
		c.JSON(200, gin.H{
			"s": 10001,
			"e": err,
		})
		return
	}

	err = transport.SetRemoteSDP(data.SDP, webrtc.SDPTypeOffer)
	sdp, err := transport.GetLocalSDP(webrtc.SDPTypeAnswer)

	if err != nil {
		fmt.Println("error", err)
		c.JSON(200, gin.H{
			"s": 10001,
			"e": err,
		})
		return
	}

	c.JSON(200, gin.H{
		"s": 10000,
		"d": map[string]string{
			"sdp": sdp,
		},
	})
}

func main() {

	go startRtmp()

	router := gin.Default()

	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowCredentials = true
	router.Use(cors.New(config))

	router.LoadHTMLFiles("./index.html")
	router.GET("/", index)
	router.POST("/rtc/v1/play", pullstream)

	router.Run(":8000")

}
