package rtp


const (
	nalTypeSlice byte = 1
	nalTypeIdr   byte = 5
	nalTypeSTAPA byte = 24
	nalTypeFuA   byte = 28
)

type H264Depacketizer struct {
	fuaFrameBuffer []byte
}

func NewH264Depacketizer() *H264Depacketizer {
	depey := &H264Depacketizer{}
	return depey
}


// TODO
func (self *H264Depacketizer) Depacket(packet []byte)  ([][]byte, bool) {
	nalTyp := packet[0] & 0x1f
	if nalTyp == nalTypeFuA {
		indicator := packet[0]
		nalHeader := packet[1]
		if (nalHeader & 0x80) == 0x80 {
			self.fuaFrameBuffer = append(self.fuaFrameBuffer, (indicator&0xE0)|(nalHeader&0x1F))
			self.fuaFrameBuffer = append(self.fuaFrameBuffer, packet[2:]...)
			return nil,false
		} else if (nalHeader & 0x40) == 0x40 {
			self.fuaFrameBuffer = append(self.fuaFrameBuffer, packet[2:]...)
			frame := make([]byte, len(self.fuaFrameBuffer))
			copy(frame,self.fuaFrameBuffer)
			self.fuaFrameBuffer = []byte{}
			return [][]byte{frame}, true
		} else {
			self.fuaFrameBuffer = append(self.fuaFrameBuffer, packet[2:]...)
			return nil, false
		}
	} else if nalTyp == nalTypeSTAPA {
		frames := [][]byte{}
		var idx uint16 = 1
		var size uint16 = (uint16)(len(packet) - 1)
		for {
			if size <= 2 {
				break
			}
			var nal_size uint16 = (uint16)(packet[idx])
			nal_size = nal_size << 8
			nal_size = nal_size | (uint16)(packet[idx+1])
			idx = idx + 2
			size = size - 2
			if nal_size <= size {
				frameBuffer := packet[idx : idx+nal_size-1]
				frame := make([]byte,len(frameBuffer))
				copy(frame,frameBuffer)
				frames = append(frames, frame)
			} else {
				break
			}

			idx += nal_size
			size -= nal_size
		}
		return frames, true
	} else {
		frame := make([]byte,len(packet))
		copy(frame,packet)
		return [][]byte{frame}, true
	}
}

