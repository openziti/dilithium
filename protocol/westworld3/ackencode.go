package westworld3

import (
	"github.com/michaelquigley/dilithium/util"
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

type ack struct {
	start int32
	end   int32
}

const ackSeriesMarker = (1 << 7)
const sequenceRangeMarker = (1 << 31)

func encodeAcks(acks []ack, data []byte) (n uint32, err error) {
	if len(acks) < 1 {
		return 0, nil
	}

	dataSz := uint32(len(data))

	if len(acks) == 1 {
		if acks[0].start == acks[0].end {
			if dataSz < 4 {
				return 0, errors.Errorf("insufficient buffer to encode ack [%d < 4]", dataSz)
			}
			util.WriteInt32(data, int32(uint32(acks[0].start) ^ sequenceRangeMarker))
			return 4, nil
		}
	}

	if len(acks) > 127 {
		return 0, errors.Errorf("ack series too large [%d > 127]", len(acks))
	}

	i := uint32(0)
	if (i + 1) > dataSz {
		return i, errors.Errorf("insufficient buffer to encode ack series [%d < %d]", dataSz, (i + 1))
	}
	data[i] = uint8(ackSeriesMarker + len(acks))
	i++

	for _, a := range acks {
		if a.start == a.end {
			if (i + 4) > dataSz {
				return i, errors.Errorf("insufficient buffer to encode ack series [%d < %d]", dataSz, i)
			}
			util.WriteInt32(data[i:i+4], int32(uint32(a.start) ^ sequenceRangeMarker))
			i += 4
		} else {
			if (i + 4) > dataSz {
				return i, errors.Errorf("insufficient buffer to encode ack series [%d < %d]", dataSz, i)
			}
			util.WriteInt32(data[i:i+4], int32(uint32(a.start) | sequenceRangeMarker))
			i += 4

			if (i + 4) > dataSz {
				return i, errors.Errorf("insufficient buffer to encode ack series [%d < %d]", dataSz, i)
			}
			util.WriteInt32(data[i:i+4], int32(uint32(a.start) ^ sequenceRangeMarker))
			i += 4
		}
	}

	return i, nil
}

func decodeAcks(data []byte) (acks []ack, err error) {
	return nil, nil
}
