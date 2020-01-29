package rtcrtmp

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/notedit/gst"
	"github.com/notedit/rtmp-lib"
	"github.com/notedit/rtmp-lib/aac"
	"github.com/notedit/rtmp-lib/av"
	"github.com/notedit/rtmp-lib/h264"
	"github.com/pion/webrtc/v2"
	uuid "github.com/satori/go.uuid"
)

type RtmpRtcStreamer struct {
	streams        []av.CodecData
	videoCodecData h264.CodecData
	audioCodecData aac.CodecData
	pipeline       *gst.Pipeline
	audiosrc       *gst.Element
	audiosink      *gst.Element
	adtsheader     []byte
	spspps         bool

	audioTrack *webrtc.Track
	videoTrack *webrtc.Track

	localSDP  string
	remoteSDP string

	streamURL string
	conn      *rtmp.Conn
	pc        *webrtc.PeerConnection
	closed    bool
}

func NewRtmpRtcStreamer(streamURL string) (*RtmpRtcStreamer, error) {

	pipeline, err := gst.ParseLaunch(aac2opus_pipeline)

	if err != nil {
		return nil, err
	}

	audiosrc := pipeline.GetByName("appsrc")
	pipeline.SetState(gst.StatePlaying)

	config := webrtc.Configuration{
		ICEServers:   []webrtc.ICEServer{},
		BundlePolicy: webrtc.BundlePolicyMaxBundle,
		SDPSemantics: webrtc.SDPSemanticsUnifiedPlan,
	}

	s := webrtc.SettingEngine{}
	s.SetConnectionTimeout(10*time.Second, 2*time.Second)
	m := webrtc.MediaEngine{}
	m.RegisterCodec(webrtc.NewRTPOpusCodec(webrtc.DefaultPayloadTypeOpus, 48000))
	m.RegisterCodec(webrtc.NewRTPH264Codec(webrtc.DefaultPayloadTypeH264, 90000))
	api := webrtc.NewAPI(webrtc.WithSettingEngine(s), webrtc.WithMediaEngine(m))

	peerConnection, err := api.NewPeerConnection(config)
	if err != nil {
		return nil, err
	}

	audioTrack, err := peerConnection.NewTrack(webrtc.DefaultPayloadTypeOpus, rand.Uint32(), uuid.NewV4().String(), uuid.NewV4().String())

	if err != nil {
		return nil, err
	}

	videoTrack, err := peerConnection.NewTrack(webrtc.DefaultPayloadTypeH264, rand.Uint32(), uuid.NewV4().String(), uuid.NewV4().String())

	if err != nil {
		return nil, err
	}

	peerConnection.AddTransceiverFromTrack(audioTrack, webrtc.RtpTransceiverInit{Direction: webrtc.RTPTransceiverDirectionSendonly})
	peerConnection.AddTransceiverFromTrack(videoTrack, webrtc.RtpTransceiverInit{Direction: webrtc.RTPTransceiverDirectionSendonly})

	rtmp2rtc := &RtmpRtcStreamer{}
	rtmp2rtc.audiosrc = audiosrc
	rtmp2rtc.pc = peerConnection
	rtmp2rtc.adtsheader = make([]byte, 7)
	rtmp2rtc.audioTrack = audioTrack
	rtmp2rtc.videoTrack = videoTrack

	peerConnection.OnConnectionStateChange(rtmp2rtc.onConnectionState)

	return rtmp2rtc, nil
}

func (r *RtmpRtcStreamer) GetLocalSDP() string {

	if r.localSDP == "" {
		sdp, _ := r.pc.CreateOffer(nil)
		r.localSDP = sdp.SDP
	}

	return r.localSDP
}

func (r *RtmpRtcStreamer) SetRemoteSDP(sdpStr string) {

	r.remoteSDP = sdpStr
	sdp := webrtc.SessionDescription{SDP: sdpStr, Type: webrtc.SDPTypeAnswer}
	r.pc.SetRemoteDescription(sdp)
}

func (r *RtmpRtcStreamer) onConnectionState(state webrtc.PeerConnectionState) {

	if state == webrtc.PeerConnectionStateConnected {
		go r.startpull()
	}

	// todo, handle other state
}

func (r *RtmpRtcStreamer) Close() {

	if r.closed {
		return
	}

	r.closed = true

	r.pc.Close()
	r.conn.Close()
	r.pipeline.SetState(gst.StateNull)
}

func (r *RtmpRtcStreamer) startpull() {

	conn, err := rtmp.Dial(r.streamURL)

	if err != nil {
		panic(err)
	}

	r.conn = conn

	r.streams, err = conn.Streams()

	if err != nil {
		panic(err)
	}

	for _, stream := range r.streams {
		if stream.Type() == av.H264 {
			r.videoCodecData = stream.(h264.CodecData)
		}
		if stream.Type() == av.AAC {
			r.audioCodecData = stream.(aac.CodecData)
		}
	}

	for {
		packet, err := conn.ReadPacket()
		if err != nil {
			break
		}
		stream := r.streams[packet.Idx]

		if stream.Type() == av.H264 {
			fmt.Println("video =====")
		} else if stream.Type() == av.AAC {

			adtsbuffer := []byte{}
			aac.FillADTSHeader(r.adtsheader, r.audioCodecData.Config, 1024, len(packet.Data))
			adtsbuffer = append(adtsbuffer, r.adtsheader...)
			adtsbuffer = append(adtsbuffer, packet.Data...)

			r.audiosrc.PushBuffer(adtsbuffer)
			fmt.Println("audio ====")
		}
	}
}
