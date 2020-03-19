package rtcrtmp

import "github.com/pion/rtp"


type RTPJitter struct {
	clockrate      uint32
	cap            uint16
	packetsCount   uint32
	nextSeqNum     uint16
	packets        []*rtp.Packet
	packetsSeqs []uint16

	lastTime uint32
	nextTime uint32

	maxWaitTime uint32
	clockInMS  uint32
}

// cap maybe 512 or 1024 or more
func NewJitter(cap uint16, clockrate uint32) *RTPJitter {
	jitter := &RTPJitter{}
	jitter.packets = make([]*rtp.Packet, cap)
	jitter.packetsSeqs = make([]uint16, cap)
	jitter.cap = cap
	jitter.clockrate = clockrate
	jitter.clockInMS = clockrate / 1000
	return jitter
}

func (self *RTPJitter) Add(packet *rtp.Packet) bool {

	idx := packet.SequenceNumber % self.cap
	self.packets[idx] = packet
	self.packetsSeqs[idx] = packet.SequenceNumber

	if self.packetsCount == 0 {
		self.nextSeqNum = packet.SequenceNumber - 1
		self.nextTime = packet.Timestamp
	}

	self.lastTime = packet.Timestamp
	self.packetsCount++
	return true
}

func (self *RTPJitter) SetMaxWaitTime(wait uint32) {
	self.maxWaitTime = wait
}


func (self *RTPJitter) GetOrdered() (out []*rtp.Packet) {
	nextSeq := self.nextSeqNum + 1
	for {
		idx := nextSeq % self.cap
		if self.packetsSeqs[idx] != nextSeq {
			// if we reach max wait time
			if (self.lastTime - self.nextTime) > self.maxWaitTime * self.clockInMS {
				nextSeq++
				continue
			}
			break
		}
		packet := self.packets[idx]
		out = append(out, packet)
		self.nextTime = packet.Timestamp
		self.nextSeqNum = nextSeq
		nextSeq++
	}
	return
}
