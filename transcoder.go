package rtcrtmp

import (
	"fmt"
	"io"
	"log"

	"github.com/3d0c/gmf"
)

const (
	aac2opus_pipeline = "appsrc do-timestamp=true is-live=true  name=appsrc ! decodebin ! audioconvert ! audioresample ! opusenc ! opusdec ! autoaudiosink "
	opus2aac_pipeline = "appsrc do-timestamp=true is-live=true  name=appsrc ! decodebin ! audioconvert ! audioresample ! opusenc "
)

type Config struct {
}

type FFTranscoder struct {
	inputFmt          *gmf.FmtCtx
	outputFmt         *gmf.FmtCtx
	inputAudioStream  *gmf.Stream
	outputAudioStream *gmf.Stream
	inputVideoStream  *gmf.Stream
	outputVideoStream *gmf.Stream
	audioCodec        *gmf.Codec
	audioCc           *gmf.CodecCtx
	videoCodec        *gmf.Codec
	videoCc           *gmf.CodecCtx

	streamURL string
}

func NewFFTranscoder(streamURL string, config *Config) *FFTranscoder {

	transcoder := &FFTranscoder{}
	transcoder.streamURL = streamURL
	return transcoder
}

func (f *FFTranscoder) Open() (err error) {

	if f.inputFmt, err = gmf.NewInputCtx(f.streamURL); err != nil {
		return
	}

	f.outputFmt = gmf.NewCtx()

	f.inputAudioStream, err = f.inputFmt.GetBestStream(gmf.AVMEDIA_TYPE_AUDIO)
	if err != nil {
		return
	}

	// audio
	f.audioCodec, err = gmf.FindEncoder("libopus")
	if err != nil {
		return
	}

	f.audioCc = gmf.NewCodecCtx(f.audioCodec)

	f.audioCc.SetSampleFmt(f.audioCc.SelectSampleFmt())
	f.audioCc.SetSampleRate(48000)
	f.audioCc.SetChannels(2)
	channelLayout := f.audioCc.SelectChannelLayout()
	f.audioCc.SetChannelLayout(channelLayout)
	f.audioCc.SetTimeBase(gmf.AVR{Num: 1, Den: 48000})

	if err = f.audioCc.Open(nil); err != nil {
		return
	}

	par := gmf.NewCodecParameters()
	if err = par.FromContext(f.audioCc); err != nil {
		return
	}
	defer par.Free()

	f.outputAudioStream, err = f.outputFmt.AddStreamWithCodeCtx(f.audioCc)
	f.outputAudioStream.SetCodecCtx(f.audioCc)

	// video
	f.inputVideoStream, err = f.inputFmt.GetBestStream(gmf.AVMEDIA_TYPE_VIDEO)
	if err != nil {
		return
	}

	par = gmf.NewCodecParameters()
	if err = par.FromContext(f.inputVideoStream.CodecCtx()); err != nil {
		panic(err)
	}
	defer par.Free()

	f.outputVideoStream, err = f.outputFmt.AddStreamWithCodeCtx(f.inputVideoStream.CodecCtx())
	if err != nil {
		return
	}

	f.outputVideoStream.CopyCodecPar(par)
	f.outputVideoStream.SetCodecCtx(f.inputVideoStream.CodecCtx())

	return
}

func (f *FFTranscoder) ReadPacket(errchan chan error) <-chan *gmf.Packet {

	packetChan := make(chan *gmf.Packet, 10)
	var pkt *gmf.Packet
	var err error
	var ist *gmf.Stream

	go func() {
		defer close(packetChan)
		for {
			if pkt, err = f.inputFmt.GetNextPacket(); err != nil {
				if err == io.EOF {
					errchan <- err
				} else {
					// todo?  how to handle this
					log.Fatalln(pkt, err)
				}
			}

			ist, err = f.inputFmt.GetStream(pkt.StreamIndex())
			if err != nil {
				errchan <- err
			}

			if ist.Type() == gmf.AVMEDIA_TYPE_VIDEO {

				if pkt.Pts() != gmf.AV_NOPTS_VALUE {
					pkt.SetPts(gmf.RescaleQRnd(pkt.Pts(), f.inputVideoStream.TimeBase(), f.outputVideoStream.TimeBase()))
				}

				if pkt.Dts() != gmf.AV_NOPTS_VALUE {
					pkt.SetDts(gmf.RescaleQRnd(pkt.Dts(), f.inputVideoStream.TimeBase(), f.outputVideoStream.TimeBase()))
				}
				fmt.Println(pkt.Time(f.outputVideoStream.TimeBase()))
				packetChan <- pkt
			}
		}
	}()

	return packetChan
}
