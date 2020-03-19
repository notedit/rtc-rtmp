package rtcrtmp

import (
	"github.com/pion/rtp"
)

type RTPBuffer struct {
	ssrc uint32
	cap  uint16

	packets     []*rtp.Packet
	packetsSeqs  []uint16
	packetsCount uint32
}

// cap can be 512
func NewRTPBuffer(cap uint16) *RTPBuffer {
	buffer := &RTPBuffer{}
	buffer.cap = cap
	buffer.packets = make([]*rtp.Packet,cap)
	buffer.packetsSeqs = make([]uint16,cap)
	return buffer
}

func (self *RTPBuffer) Add(packet *rtp.Packet) {
	if self.ssrc == 0 {
		self.ssrc = packet.SSRC
	}

	idx := packet.SequenceNumber % self.cap
	self.packets[idx] = packet
	self.packetsSeqs[idx] = packet.SequenceNumber
	self.packetsCount++
}

func (self *RTPBuffer) Get(seq uint16) *rtp.Packet {
	idx := seq % self.cap
	return self.packets[idx]
}
