package rtcrtmp

import (
	"fmt"
	"testing"
)

type RTCSource struct {
	extSeqNum uint32
	cycles  uint32
}


func (self *RTCSource) SetSeqNum(seqNum uint16) uint32 {

	if seqNum < 0x0FFF && (self.extSeqNum & 0xFFFF) > 0xF000 {
		self.cycles++
	}

	extSeqNum := self.cycles << 16 | uint32(seqNum)

	if extSeqNum > self.extSeqNum {
		self.extSeqNum = extSeqNum
	}

	return  self.cycles
}


func TestSetSeqNum(t *testing.T) {

	seqs := []uint16{65532,65533,65534,1,65535,2,3,4}
	source := &RTCSource{}

	for _,seq := range seqs {
		cycles := source.SetSeqNum(seq)
		fmt.Println(cycles << 16 | uint32(seq))
	}
}