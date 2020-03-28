module github.com/notedit/rtc-rtmp

go 1.13

require (
	github.com/gin-contrib/cors v1.3.0
	github.com/gin-gonic/gin v1.5.0
	github.com/notedit/go-fdkaac v0.0.0-20200307100649-833bc3aabc30 // indirect
	github.com/notedit/rtmp-lib v0.0.7
	github.com/pion/rtcp v1.2.1
	github.com/pion/rtp v1.3.2
	github.com/pion/webrtc/v2 v2.2.3
	github.com/satori/go.uuid v1.2.0
	layeh.com/gopus v0.0.0-20161224163843-0ebf989153aa
)

replace github.com/notedit/rtmp-lib v0.0.7 => ../rtmp-lib
