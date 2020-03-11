package rtcrtmp

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
)

func NaluType(firstByte byte) NalUnitType {
	return NalUnitType((firstByte & 0x1F) >> 0)
}


