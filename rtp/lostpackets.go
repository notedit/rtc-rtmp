package rtp

import (
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"time"
)

const kMaxNackNumber uint16 = 512
const kDefaultWaitNackTime = 5
const kDefaultMaxRetry uint8 = 20

type NackInfo struct {
	seqNum     uint16
	sentTimeNs int64
	retries    uint8
}

type RTPLostPackets struct {
	mediaSSRC    uint32
	seqNums      []uint16
	nackList     []NackInfo
	latestSeq    uint16
	lastNackTime int64
}

func NewRTPLostPackets() *RTPLostPackets {
	lost := &RTPLostPackets{}
	lost.seqNums = make([]uint16, kMaxNackNumber)
	lost.nackList = make([]NackInfo, 0)
	return lost
}

func (self *RTPLostPackets) AddPacket(packet *rtp.Packet) int {

	if self.mediaSSRC == 0 {
		self.mediaSSRC = packet.SSRC
		self.latestSeq = packet.SequenceNumber
	}

	index := packet.SequenceNumber % kMaxNackNumber
	self.seqNums[index] = packet.SequenceNumber

	if packet.SequenceNumber <= self.latestSeq {
		return 0
	}

	if packet.SequenceNumber > self.latestSeq && (packet.SequenceNumber-self.latestSeq) > 0x0fff {
		return 0
	}

	now := time.Now().UnixNano()
	for seq := self.latestSeq + 1; seq < packet.SequenceNumber; seq++ {
		self.nackList = append(self.nackList, NackInfo{seq, now, 0})
	}

	lossNum := int(packet.SequenceNumber - self.latestSeq - 1)
	return lossNum
}

func (self *RTPLostPackets) GetNacks(nowNs int64) []rtcp.NackPair {

	if self.lastNackTime == 0 {
		self.lastNackTime = nowNs
		return nil
	}

	if len(self.nackList) > 0 && nowNs > (self.lastNackTime + 20 * 1e6) {

		nacks := []rtcp.NackPair{}
		nackList := self.nackList

		var pairBaseSeq uint16
		var pairInited bool
		var mask rtcp.PacketBitmap

		for _,nack := range nackList {
			idx := nack.seqNum % kMaxNackNumber
			if nack.seqNum == self.seqNums[idx] || (nack.retries == 0 && nack.sentTimeNs+kDefaultWaitNackTime*1e6 > nowNs) ||
				nack.retries > kDefaultMaxRetry {
				continue
			}
			if !pairInited {
				pairInited = true
				pairBaseSeq = nack.seqNum
				nack.sentTimeNs = nowNs
				nack.retries++
				self.nackList = append(self.nackList,nack)
			} else {
				delta := nack.seqNum - pairBaseSeq - 1
				nack.sentTimeNs = nowNs
				nack.retries++
				self.nackList = append(self.nackList,nack)
				if delta > 15 {
					nacks = append(nacks, rtcp.NackPair{PacketID: pairBaseSeq, LostPackets: mask})
					pairInited = false
					pairBaseSeq = 0
					mask = 0
				} else {
					mask |= 1 << delta
				}

			}
		}

		if pairBaseSeq != 0 {
			nacks = append(nacks, rtcp.NackPair{PacketID: pairBaseSeq, LostPackets: mask})
		}
		self.lastNackTime = nowNs

		return nacks
	}

	return nil
}

