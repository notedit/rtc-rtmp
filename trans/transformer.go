package trans

import (
	"github.com/notedit/rtmp-lib/audio"
	"github.com/notedit/rtmp-lib/av"
	"time"
)

type Transformer struct {
	inSampleFormat   av.SampleFormat
	outSampleFormat  av.SampleFormat
	inChannelLayout  av.ChannelLayout
	outChannelLayout av.ChannelLayout
	outbitrate       int
	inSampleRate     int
	outSampleRate    int
	enc              av.AudioEncoder
	dec              av.AudioDecoder
	timeline         *av.Timeline
}

func (t *Transformer) Setup() error {
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
	enc.SetBitrate(t.outbitrate)
	err = enc.Setup()
	if err != nil {
		return err
	}
	t.enc = enc
	t.timeline = &av.Timeline{}
	return nil
}

func (t *Transformer) SetInSampleRate(samplerate int) error {
	t.inSampleRate = samplerate
	return nil
}

func (t *Transformer) SetInChannelLayout(channel av.ChannelLayout) error {
	t.inChannelLayout = channel
	return nil
}

func (t *Transformer) SetInSampleFormat(sampleformat av.SampleFormat) error {
	t.inSampleFormat = sampleformat
	return nil
}

func (t *Transformer) SetOutSampleRate(samplerate int) error {
	t.outSampleRate = samplerate
	return nil
}

func (t *Transformer) SetOutChannelLayout(channel av.ChannelLayout) error {
	t.outChannelLayout = channel
	return nil
}

func (t *Transformer) SetOutSampleFormat(sampleformat av.SampleFormat) error {
	t.outSampleFormat = sampleformat
	return nil
}

func (t *Transformer) SetOutBitrate(bitrate int) error {
	t.outbitrate = bitrate
	return nil
}

func (t *Transformer) Do(pkt av.Packet) (out []av.Packet, err error) {

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
		outpkt := av.Packet{Idx: pkt.Idx, Data: _outpkt}
		outpkt.Time = t.timeline.Pop(dur)
		out = append(out, outpkt)
	}
	return
}

func (t *Transformer) Close() {
	t.enc.Close()
	t.dec.Close()
}
