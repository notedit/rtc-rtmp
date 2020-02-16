package rtcrtmp

import "strconv"

type NalUnitType uint8

const ( //   Table 7-1 NAL unit type codes
	Unspecified              NalUnitType = 0  // Unspecified
	CodedSliceNonIdr         NalUnitType = 1  // Coded slice of a non-IDR picture
	CodedSliceDataPartitionA NalUnitType = 2  // Coded slice data partition A
	CodedSliceDataPartitionB NalUnitType = 3  // Coded slice data partition B
	CodedSliceDataPartitionC NalUnitType = 4  // Coded slice data partition C
	CodedSliceIdr            NalUnitType = 5  // Coded slice of an IDR picture
	SEI                      NalUnitType = 6  // Supplemental enhancement information (SEI)
	SPS                      NalUnitType = 7  // Sequence parameter set
	PPS                      NalUnitType = 8  // Picture parameter set
	AUD                      NalUnitType = 9  // Access unit delimiter
	EndOfSequence            NalUnitType = 10 // End of sequence
	EndOfStream              NalUnitType = 11 // End of stream
	Filler                   NalUnitType = 12 // Filler data
	SpsExt                   NalUnitType = 13 // Sequence parameter set extension
	// 14..18           // Reserved
	NalUnitTypeCodedSliceAux NalUnitType = 19 // Coded slice of an auxiliary coded picture without partitioning
	// 20..23           // Reserved
	// 24..31           // Unspecified
)

func NalUnitTypeStr(v NalUnitType) string {
	str := "Unknown"
	switch v {
	case 0:
		{
			str = "Unspecified"
		}
	case 1:
		{
			str = "CodedSliceNonIdr"
		}
	case 2:
		{
			str = "CodedSliceDataPartitionA"
		}
	case 3:
		{
			str = "CodedSliceDataPartitionB"
		}
	case 4:
		{
			str = "CodedSliceDataPartitionC"
		}
	case 5:
		{
			str = "CodedSliceIdr"
		}
	case 6:
		{
			str = "SEI"
		}
	case 7:
		{
			str = "SPS"
		}
	case 8:
		{
			str = "PPS"
		}
	case 9:
		{
			str = "AUD"
		}
	case 10:
		{
			str = "EndOfSequence"
		}
	case 11:
		{
			str = "EndOfStream"
		}
	case 12:
		{
			str = "Filler"
		}
	case 13:
		{
			str = "SpsExt"
		}
	case 19:
		{
			str = "NalUnitTypeCodedSliceAux"
		}
	default:
		{
			str = "Unknown"
		}
	}
	str = str + "(" + strconv.FormatInt(int64(v), 10) + ")"
	return str
}

type Nal struct {
	PictureOrderCount uint32

	// NAL header
	ForbiddenZeroBit bool
	RefIdc           uint8
	UnitType         NalUnitType

	Data []byte // header byte + rbsp
}

func NewNal() Nal {
	return Nal{PictureOrderCount: 0, ForbiddenZeroBit: false, RefIdc: 0, UnitType: Unspecified, Data: make([]byte, 0)}
}

func (h *Nal) ParseHeader(firstByte byte) {
	h.ForbiddenZeroBit = (((firstByte & 0x80) >> 7) == 1) // 0x80 = 0b10000000
	h.RefIdc = (firstByte & 0x60) >> 5                    // 0x60 = 0b01100000
	h.UnitType = NalUnitType((firstByte & 0x1F) >> 0)     // 0x1F = 0b00011111
}
