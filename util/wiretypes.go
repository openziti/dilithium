package util

func ReadInt64(buf []byte) (v int64) {
	v |= int64(buf[0]) << 56
	v |= int64(buf[1]) << 48
	v |= int64(buf[2]) << 40
	v |= int64(buf[3]) << 32
	v |= int64(buf[4]) << 24
	v |= int64(buf[5]) << 16
	v |= int64(buf[6]) << 8
	v |= int64(buf[7])
	return
}

func WriteInt64(buf []byte, v int64) {
	buf[0] = byte(v >> 56)
	buf[1] = byte(v >> 48)
	buf[2] = byte(v >> 40)
	buf[3] = byte(v >> 32)
	buf[4] = byte(v >> 24)
	buf[5] = byte(v >> 16)
	buf[6] = byte(v >> 8)
	buf[7] = byte(v)
}

func ReadUint32(buf []byte) (v uint32) {
	v |= uint32(buf[0]) << 24
	v |= uint32(buf[1]) << 16
	v |= uint32(buf[2]) << 8
	v |= uint32(buf[3])
	return
}

func WriteUint32(buf []byte, v uint32) {
	buf[0] = byte(v >> 24)
	buf[1] = byte(v >> 16)
	buf[2] = byte(v >> 8)
	buf[3] = byte(v)
}

func ReadInt32(buf []byte) (v int32) {
	v |= int32(buf[0]) << 24
	v |= int32(buf[1]) << 16
	v |= int32(buf[2]) << 8
	v |= int32(buf[3])
	return
}

func WriteInt32(buf []byte, v int32) {
	buf[0] = byte(v >> 24)
	buf[1] = byte(v >> 16)
	buf[2] = byte(v >> 8)
	buf[3] = byte(v)
}

func ReadUint16(buf []byte) (v uint16) {
	v |= uint16(buf[0]) << 8
	v |= uint16(buf[1])
	return
}

func WriteUint16(buf []byte, v uint16) {
	buf[0] = byte(v >> 8)
	buf[1] = byte(v)
}
