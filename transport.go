package rtcrtmp

import (
	"fmt"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	rtputil "github.com/notedit/rtc-rtmp/rtp"
	"github.com/pion/webrtc/v2"
	"github.com/rs/zerolog/log"
	uuid "github.com/satori/go.uuid"
	"io"
	"sync"
	"time"
)

type RTCTransport struct {
	id         string
	media      webrtc.MediaEngine
	api        *webrtc.API
	pc         *webrtc.PeerConnection
	videoTrack *webrtc.Track
	audioTrack *webrtc.Track

	videoBuffer *rtputil.RTPBuffer
	audioBuffer *rtputil.RTPBuffer

	connected bool

	endpoint  string
	localsdp  string
	remotesdp string
	stop      bool
	sync.RWMutex
}

func NewRTCTransport(id string, endpoint string) (*RTCTransport, error) {

	rtcpfb := []webrtc.RTCPFeedback{
		webrtc.RTCPFeedback{
			Type: webrtc.TypeRTCPFBTransportCC,
		},
		webrtc.RTCPFeedback{
			Type: webrtc.TypeRTCPFBCCM,
		},
		webrtc.RTCPFeedback{
			Type: webrtc.TypeRTCPFBNACK,
		},
	}

	s := webrtc.SettingEngine{}
	s.SetConnectionTimeout(10*time.Second, 2*time.Second)
	s.SetLite(true)
	s.SetTrickle(false)
	ips := []string{endpoint}
	s.SetNAT1To1IPs(ips, webrtc.ICECandidateTypeHost)

	m := webrtc.MediaEngine{}
	m.RegisterCodec(webrtc.NewRTPOpusCodec(webrtc.DefaultPayloadTypeOpus, 48000))
	m.RegisterCodec(webrtc.NewRTPH264CodecExt(webrtc.DefaultPayloadTypeH264, 90000, rtcpfb))
	api := webrtc.NewAPI(webrtc.WithSettingEngine(s), webrtc.WithMediaEngine(m))

	config := webrtc.Configuration{
		ICEServers:   []webrtc.ICEServer{},
		BundlePolicy: webrtc.BundlePolicyMaxBundle,
		SDPSemantics: webrtc.SDPSemanticsUnifiedPlan,
	}

	pc, _ := api.NewPeerConnection(config)

	transport := &RTCTransport{
		id:       id,
		media:    m,
		api:      api,
		pc:       pc,
		endpoint: endpoint,
	}

	streamID := uuid.NewV4().String()
	audioTrack, err := pc.NewTrack(webrtc.DefaultPayloadTypeOpus, DefaultOpusSSRC, uuid.NewV4().String(), streamID)

	if err != nil {
		return nil, err
	}

	videoTrack, err := pc.NewTrack(webrtc.DefaultPayloadTypeH264, DefaultH264SSRC, uuid.NewV4().String(), streamID)

	if err != nil {
		return nil, err
	}

	pc.OnConnectionStateChange(transport.onConnectionState)

	pc.AddTransceiverFromTrack(audioTrack, webrtc.RtpTransceiverInit{Direction: webrtc.RTPTransceiverDirectionSendonly})
	videoTransceiver, _ := pc.AddTransceiverFromTrack(videoTrack, webrtc.RtpTransceiverInit{Direction: webrtc.RTPTransceiverDirectionSendonly})

	transport.audioTrack = audioTrack
	transport.videoTrack = videoTrack

	transport.videoBuffer = rtputil.NewRTPBuffer(512)
	transport.audioBuffer = rtputil.NewRTPBuffer(512)

	//transport.handleRTCP(audioTransceiver.Sender())
	transport.handleRTCP(videoTransceiver.Sender())

	return transport, nil
}

func (self *RTCTransport) ID() string {
	return self.id
}

func (self *RTCTransport) GetLocalSDP(sdpType webrtc.SDPType) (string, error) {

	var sdp webrtc.SessionDescription
	var err error

	if self.localsdp == "" {
		if sdpType == webrtc.SDPTypeOffer {
			sdp, err = self.pc.CreateOffer(nil)

		} else {
			sdp, err = self.pc.CreateAnswer(nil)
		}
		err = self.pc.SetLocalDescription(sdp)
		self.localsdp = sdp.SDP
	}

	return self.localsdp, err
}

func (self *RTCTransport) SetRemoteSDP(sdpstr string, sdpType webrtc.SDPType) error {

	if self.pc == nil {
		return fmt.Errorf("peerconnection does not init yet")
	}

	self.remotesdp = sdpstr
	sdp := webrtc.SessionDescription{SDP: sdpstr, Type: sdpType}
	err := self.pc.SetRemoteDescription(sdp)

	return err
}

func (self *RTCTransport) WriteRTP(packet *rtp.Packet) (err error) {

	if !self.connected {
		fmt.Println("transport does not connected ========")
		return
	}

	if packet.SSRC == DefaultOpusSSRC {
		self.audioBuffer.Add(packet)
		err = self.audioTrack.WriteRTP(packet)
	} else if packet.SSRC == DefaultH264SSRC {
		self.videoBuffer.Add(packet)
		err = self.videoTrack.WriteRTP(packet)
	} else {
		err = fmt.Errorf("ssrc does not exist")
	}
	return
}

func (self *RTCTransport) Stop() (err error) {
	if self.stop {
		return
	}
	err = self.pc.Close()
	self.stop = true
	return
}


func (self *RTCTransport) handleRTCP(sender *webrtc.RTPSender) {
	go func() {
		for {
			if self.stop {
				return
			}
			pkts, err := sender.ReadRTCP()
			if err != nil {
				if err == io.EOF {
					return
				}
			}
			for _, pkt := range pkts {
				switch pkt.(type) {
				case *rtcp.TransportLayerNack:
					nack := pkt.(*rtcp.TransportLayerNack)
					//log.Debug().Msg(nack.String())
					for _, nackPair := range nack.Nacks {

						fmt.Println("nack  ====", nackPair.PacketList())

						for _,seq := range nackPair.PacketList() {
							rtpPkt := self.videoBuffer.Get(seq)
							if rtpPkt != nil {
								//log.Debug().Msgf("ssrc %d  packet seq %d", nack.SenderSSRC, nackPair.LostPackets())
								self.videoTrack.WriteRTP(rtpPkt)
								continue
							}
							log.Debug().Msgf("rtp buffer can not find  %d", seq)
						}
					}
				case *rtcp.PictureLossIndication:
					pli := pkt.(*rtcp.PictureLossIndication)
					log.Debug().Msg(pli.String())
				case *rtcp.ReceiverReport:
					//report := pkt.(*rtcp.ReceiverReport)
					//log.Debug().Msg(report.String())
				}
			}
		}
	}()
}

func (self *RTCTransport) onConnectionState(state webrtc.PeerConnectionState) {
	
	if state == webrtc.PeerConnectionStateConnected {
		self.connected = true
		log.Debug().Msg("peerconnection connected")
	}
}
