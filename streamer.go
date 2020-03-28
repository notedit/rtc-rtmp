package rtcrtmp

import (
	"bytes"
	"fmt"
	"time"

	"github.com/notedit/rtc-rtmp/transformer"
	"github.com/notedit/rtmp-lib"
	"github.com/notedit/rtmp-lib/aac"
	"github.com/notedit/rtmp-lib/av"
	"github.com/notedit/rtmp-lib/h264"
	"github.com/pion/webrtc/v2"
	uuid "github.com/satori/go.uuid"
)

var naluHeader = []byte{0, 0, 0, 1}


type RtmpStreamer struct {
	streams    []av.CodecData
	videoCodec h264.CodecData
	audioCodec aac.CodecData
	adtsheader []byte
	spspps     bool

	transform *transformer.Transformer

	audioTrack *webrtc.Track
	videoTrack *webrtc.Track

	localSDP  string
	remoteSDP string

	lastVideoTime time.Duration
	lastAudioTime time.Duration

	streamURL string
	conn      *rtmp.Conn
	pc        *webrtc.PeerConnection
	closed    bool
}

func NewRtmpRtcStreamer(streamURL string) (*RtmpStreamer, error) {

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

	streamID := uuid.NewV4().String()
	audioTrack, err := peerConnection.NewTrack(webrtc.DefaultPayloadTypeOpus, 333, uuid.NewV4().String(), streamID)

	if err != nil {
		return nil, err
	}

	videoTrack, err := peerConnection.NewTrack(webrtc.DefaultPayloadTypeH264, 666, uuid.NewV4().String(), streamID)

	if err != nil {
		return nil, err
	}

	peerConnection.AddTransceiverFromTrack(audioTrack, webrtc.RtpTransceiverInit{Direction: webrtc.RTPTransceiverDirectionSendonly})
	peerConnection.AddTransceiverFromTrack(videoTrack, webrtc.RtpTransceiverInit{Direction: webrtc.RTPTransceiverDirectionSendonly})

	transform := &transformer.Transformer{}

	streamer := &RtmpStreamer{}
	streamer.pc = peerConnection
	streamer.audioTrack = audioTrack
	streamer.videoTrack = videoTrack
	streamer.streamURL = streamURL
	streamer.transform = transform

	peerConnection.OnConnectionStateChange(streamer.onConnectionState)

	return streamer, nil
}

func (r *RtmpStreamer) GetLocalSDP(sdpType webrtc.SDPType) (string, error) {

	var sdp webrtc.SessionDescription
	var err error

	if r.localSDP == "" {
		if sdpType == webrtc.SDPTypeOffer {
			sdp, err = r.pc.CreateOffer(nil)

		} else {
			sdp, err = r.pc.CreateAnswer(nil)
		}
		err = r.pc.SetLocalDescription(sdp)
		r.localSDP = sdp.SDP
	}

	return r.localSDP, err
}

func (r *RtmpStreamer) SetRemoteSDP(sdpStr string, sdpType webrtc.SDPType) error {

	r.remoteSDP = sdpStr
	sdp := webrtc.SessionDescription{SDP: sdpStr, Type: sdpType}
	err := r.pc.SetRemoteDescription(sdp)

	return err
}


func (r *RtmpStreamer) onConnectionState(state webrtc.PeerConnectionState) {

	if state == webrtc.PeerConnectionStateConnected {
		go r.PullStream()
	}

	// todo, handle other state
}

func (r *RtmpStreamer) Close() {

	if r.closed {
		return
	}

	r.closed = true

	r.pc.Close()
	r.conn.Close()
	r.transform.Close()
}

func (r *RtmpStreamer) PullStream() {

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
			r.videoCodec = stream.(h264.CodecData)
		}
		if stream.Type() == av.AAC {
			r.audioCodec = stream.(aac.CodecData)
			r.transform.SetInSampleRate(r.audioCodec.SampleRate())
			r.transform.SetInChannelLayout(r.audioCodec.ChannelLayout())
			r.transform.SetInSampleFormat(r.audioCodec.SampleFormat())
			r.transform.SetOutChannelLayout(av.CH_STEREO)
			r.transform.SetOutSampleRate(48000)
			r.transform.SetOutSampleFormat(av.S16)
			r.transform.Setup()
		}
	}

	for {
		packet, err := conn.ReadPacket()
		if err != nil {
			break
		}

		stream := r.streams[packet.Idx]

		if stream.Type().IsVideo() {
			var samples uint32
			if r.lastVideoTime == 0 {
				samples = 0
			} else {
				samples = uint32(uint64(packet.Time-r.lastVideoTime) * 90000 / 1000000000)
			}

			var b bytes.Buffer
			// todo  may check the sps and ppt packet
			if packet.IsKeyFrame {
				b.Write(naluHeader)
				b.Write(r.videoCodec.SPS())
				b.Write(naluHeader)
				b.Write(r.videoCodec.PPS())
			}

			if packet.Data[0] == 0x00 && packet.Data[1] == 0x00 && packet.Data[2] == 0x00 && packet.Data[3] == 0x01 {
				fmt.Println("0001 prefix")
				b.Write(packet.Data)
			} else {
				nalus, _ := h264.SplitNALUs(packet.Data)
				for _, nalu := range nalus {
					b.Write(naluHeader)
					b.Write(nalu)
				}
			}

			packets := r.videoTrack.Packetizer().Packetize(b.Bytes(), samples)
			for _, p := range packets {
				err := r.videoTrack.WriteRTP(p)
				if err != nil {
					fmt.Println(err)
					continue
				}
			}
			r.lastVideoTime = packet.Time

		} else if stream.Type() == av.AAC {

			pkts,err := r.transform.Do(packet)
			if err != nil {
				fmt.Println("transform error",err)
				continue
			}

			for _,pkt := range pkts {
				packets := r.audioTrack.Packetizer().Packetize(pkt.Data, 960)
				for _, p := range packets {
					err := r.audioTrack.WriteRTP(p)
					if err != nil {
						fmt.Println(err)
						continue
					}
				}

				r.lastAudioTime = pkt.Time
				//r.audioTrack.WriteSample(media.Sample{Data: pkt.Data, Samples: 960})
			}
		}
	}
}

