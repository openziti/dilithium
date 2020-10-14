package westworld3

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

func encodeAcks(acks []ack, data []byte) (n uint32, err error) {
	return 0, nil
}

func decodeAcks(data []byte) (acks []ack, err error) {
	return nil, nil
}