package transformer

import (
	"fmt"
	"github.com/notedit/rtmp-lib/audio"
	"github.com/notedit/rtmp-lib/av"
	"time"
)

import "C"




type FFTransformer struct {
	inSampleFormat   av.SampleFormat
	outSampleFormat  av.SampleFormat
	inChannelLayout  av.ChannelLayout
	outChannelLayout av.ChannelLayout
	inSampleRate     int
	outSampleRate    int
	enc              av.AudioEncoder
	dec              av.AudioDecoder
	timeline         *av.Timeline
}

func (t *FFTransformer) Setup() error {
	dec, err := audio.NewAudioDecoderByName("aac")
	if err != nil {
		return err
	}
	dec.SetSampleRate(t.inSampleRate)
	dec.SetSampleFormat(t.inSampleFormat)
	dec.SetChannelLayout(t.inChannelLayout)
	err = dec.Setup()
	if err != nil {
		return err
	}
	t.dec = dec
	enc, err := audio.NewAudioEncoderByName("libopus")
	if err != nil {
		return err
	}
	enc.SetSampleRate(t.outSampleRate)
	enc.SetSampleFormat(t.outSampleFormat)
	enc.SetChannelLayout(t.outChannelLayout)
	enc.SetBitrate(64000)
	err = enc.Setup()
	if err != nil {
		return err
	}
	t.enc = enc
	t.timeline = &av.Timeline{}
	return nil
}

func (t *FFTransformer) SetInSampleRate(samplerate int) error {
	t.inSampleRate = samplerate
	return nil
}

func (t *FFTransformer) SetInChannelLayout(channel av.ChannelLayout) error {
	t.inChannelLayout = channel
	return nil
}

func (t *FFTransformer) SetInSampleFormat(sampleformat av.SampleFormat) error {
	t.inSampleFormat = sampleformat
	return nil
}

func (t *FFTransformer) SetOutSampleRate(samplerate int) error {
	t.outSampleRate = samplerate
	return nil
}

func (t *FFTransformer) SetOutChannelLayout(channel av.ChannelLayout) error {
	t.outChannelLayout = channel
	return nil
}

func (t *FFTransformer) SetOutSampleFormat(sampleformat av.SampleFormat) error {
	t.outSampleFormat = sampleformat
	return nil
}

func (t *FFTransformer) Do(pkt av.Packet) (out []av.Packet, err error) {

	var dur time.Duration
	var frame av.AudioFrame
	var ok bool
	if ok,frame,err = t.dec.Decode(pkt.Data); err != nil {
		return
	}


	if !ok {
		return
	}

	if dur,err = t.dec.PacketDuration(pkt.Data); err != nil {
		return
	}

	t.timeline.Push(pkt.Time,dur)

	var _outpkts [][]byte
	if _outpkts, err = t.enc.Encode(frame); err != nil {
		return
	}

	for _, _outpkt := range _outpkts {
		if dur, err = t.enc.PacketDuration(_outpkt); err != nil {
			return
		}
		fmt.Println("opus dur", dur)
		outpkt := av.Packet{Idx: pkt.Idx, Data: _outpkt}
		outpkt.Time = t.timeline.Pop(dur)
		out = append(out, outpkt)
	}
	return
}

func (t *FFTransformer) Close() {
	t.enc.Close()
	t.dec.Close()
}
