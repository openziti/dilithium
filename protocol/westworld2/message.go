package westworld2

import (
	"github.com/pkg/errors"
)

type wireMessage struct {
	seq    int32
	mt     messageType
	ack    int32
	data   []byte
	buffer *buffer
}

func newHello(seq int32, buffer *buffer) *wireMessage {
	wm := &wireMessage{
		seq:    seq,
		mt:     HELLO,
		ack:    -1,
		buffer: buffer,
	}
	return wm.encode()
}

func newHelloAck(seq, ack int32, buffer *buffer) *wireMessage {
	wm := &wireMessage{
		seq:    seq,
		mt:     HELLO,
		ack:    ack,
		buffer: buffer,
	}
	return wm.encode()
}

func newData(seq int32, data []byte, buffer *buffer) *wireMessage {
	n := copy(buffer.data[11:], data)
	wm := &wireMessage{
		seq:    seq,
		mt:     DATA,
		ack:    -1,
		data:   buffer.data[11:11+n],
		buffer: buffer,
	}
	return wm.encode()
}

func newAck(seqFor int32, buffer *buffer) *wireMessage {
	wm := &wireMessage{
		seq:    -1,
		mt:     ACK,
		ack:    seqFor,
		buffer: buffer,
	}
	return wm.encode()
}

func (self *wireMessage) encode() *wireMessage {
	WriteInt32(self.buffer.data[0:4], self.seq)
	self.buffer.data[4] = byte(self.mt)
	WriteInt32(self.buffer.data[5:9], self.ack)
	WriteUint16(self.buffer.data[9:11], uint16(len(self.data)))
	self.buffer.sz = uint16(11 + len(self.data))
	return self
}

func decode(buffer *buffer) (*wireMessage, error) {
	dataLen := ReadUint16(buffer.data[9:11])
	if 11+dataLen > buffer.sz {
		return nil, errors.Errorf("short buffer [%d != %d]", 11+dataLen, buffer.sz)
	}
	wm := &wireMessage{
		seq:    ReadInt32(buffer.data[0:4]),
		mt:     messageType(buffer.data[4]),
		ack:    ReadInt32(buffer.data[5:9]),
		data:   buffer.data[11:11+dataLen],
		buffer: buffer,
	}
	return wm, nil
}

type messageType uint8

const (
	HELLO messageType = iota
	ACK
	DATA
	CLOSE
)

func ReadInt32(buf []byte) (v int32) {
	v |= int32(buf[0])
	v |= int32(buf[1]) << 8
	v |= int32(buf[2]) << 16
	v |= int32(buf[3]) << 24
	return
}

func WriteInt32(buf []byte, v int32) {
	buf[0] = byte(v)
	buf[1] = byte(v >> 8)
	buf[2] = byte(v >> 16)
	buf[3] = byte(v >> 24)
}

func ReadUint16(buf []byte) (v uint16) {
	v |= uint16(buf[0])
	v |= uint16(buf[1]) << 8
	return
}

func WriteUint16(buf []byte, v uint16) {
	buf[0] = byte(v)
	buf[1] = byte(v >> 8)
}
