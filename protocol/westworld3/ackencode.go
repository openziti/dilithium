package westworld3

import (
	"github.com/openziti/dilithium/util"
	"github.com/pkg/errors"
)

/*
 * ACK Encoding Format
 *
 * If the high-bit of the first byte of the ACK region is low, then this is a single int32 containing a sequence number.
 *
 * If the high-bit of the first byte of the ACK region is high, then we know we're dealing with multiple ACKs (or ACK
 * ranges) encoded in series. The remaining 7 bits contain the number of ACKs (or ACK ranges) encoded in the series.
 *
 * When decoding an ACK from a series, we use the high bit of the 4-byte int32 to determine if this is an ACK range.
 * If the high-bit is set, then we know to expect that there are actually 2 int32s in a row, definining the lower and
 * upper bounds of the range. If the high-bit is low, then we know this is a single sequence number.
 */

type Ack struct {
	Start int32
	End   int32
}

const ackSeriesMarker = uint8(1 << 7)
const sequenceRangeMarker = uint32(1 << 31)
const sequenceRangeInvert = 0xFFFFFFFF ^ sequenceRangeMarker

func EncodeAcks(acks []Ack, data []byte) (n uint32, err error) {
	if len(acks) < 1 {
		return 0, nil
	}
	if len(acks) > 127 {
		return 0, errors.Errorf("ack series too large [%d > 127]", len(acks))
	}

	dataSz := uint32(len(data))

	if len(acks) == 1 && acks[0].Start == acks[0].End {
		if dataSz < 4 {
			return 0, errors.Errorf("insufficient buffer to encode ack [%d < 4]", dataSz)
		}
		util.WriteInt32(data, int32(uint32(acks[0].Start)&sequenceRangeInvert))
		return 4, nil
	}

	i := uint32(0)
	if (i + 1) > dataSz {
		return i, errors.Errorf("insufficient buffer to encode ack series [%d < %d]", dataSz, i+1)
	}
	data[i] = ackSeriesMarker + uint8(len(acks))
	i++

	for _, a := range acks {
		if a.Start == a.End {
			if (i + 4) > dataSz {
				return i, errors.Errorf("insufficient buffer to encode ack series [%d < %d]", dataSz, i)
			}
			util.WriteInt32(data[i:i+4], int32(uint32(a.Start)&sequenceRangeInvert))
			i += 4

		} else {
			if (i + 4) > dataSz {
				return i, errors.Errorf("insufficient buffer to encode ack series [%d < %d]", dataSz, i)
			}
			util.WriteInt32(data[i:i+4], int32(uint32(a.Start)|sequenceRangeMarker))
			i += 4

			if (i + 4) > dataSz {
				return i, errors.Errorf("insufficient buffer to encode ack series [%d < %d]", dataSz, i)
			}
			util.WriteInt32(data[i:i+4], int32(uint32(a.End)&sequenceRangeInvert))
			i += 4
		}
	}

	return i, nil
}

func DecodeAcks(data []byte) (acks []Ack, sz uint32, err error) {
	dataSz := uint32(len(data))
	if dataSz < 4 {
		return nil, 0, errors.Errorf("short ack buffer [%d < 4]", dataSz)
	}

	if data[0]&ackSeriesMarker == 0 {
		seq := util.ReadInt32(data[0:4])
		acks = append(acks, Ack{seq, seq})
		return acks, 4, nil

	} else {
		seriesSz := int(data[0] ^ ackSeriesMarker)
		sz = 1
		for i := 0; i < seriesSz; i++ {
			first := util.ReadInt32(data[sz : sz+4])
			if uint32(first)&sequenceRangeMarker == sequenceRangeMarker {
				sz += 4
				second := util.ReadInt32(data[sz : sz+4])
				acks = append(acks, Ack{int32(uint32(first) & sequenceRangeInvert), int32(uint32(second) & sequenceRangeInvert)})

			} else {
				acks = append(acks, Ack{int32(uint32(first) & sequenceRangeInvert), int32(uint32(first) & sequenceRangeInvert)})
			}
			sz += 4
		}
	}
	return
}
