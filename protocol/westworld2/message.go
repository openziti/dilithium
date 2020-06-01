package westworld2

import (
	"github.com/pkg/errors"
	"net"
)

type wireMessage struct {
	seq    int32
	mt     messageType
	ack    int32
	data   []byte
	buffer *buffer
}

func readWireMessage(conn *net.UDPConn, pool *pool) (wm *wireMessage, peer *net.UDPAddr, err error) {
	buffer := pool.get()
	var n int
	n, peer, err = conn.ReadFromUDP(buffer.data)
	if err != nil {
		return nil, peer, errors.Wrap(err, "peer read")
	}
	buffer.sz = uint16(n)

	wm, err = decode(buffer)
	if err != nil {
		return nil, peer, errors.Wrap(err, "decode")
	}

	return
}

func writeWireMessage(wm *wireMessage, conn *net.UDPConn, peer *net.UDPAddr) error {
	n, err := conn.WriteToUDP(wm.buffer.data[:wm.buffer.sz], peer)
	if err != nil {
		return errors.Wrap(err, "peer write")
	}
	if uint16(n) != wm.buffer.sz {
		return errors.Errorf("short peer write [%d != %d]", n, wm.buffer.sz)
	}
	return nil
}

func newHello(seq int32, pool *pool) *wireMessage {
	wm := &wireMessage{
		seq:    seq,
		mt:     HELLO,
		ack:    -1,
		buffer: pool.get(),
	}
	return wm.encode()
}

func newHelloAck(seq, ack int32, pool *pool) *wireMessage {
	wm := &wireMessage{
		seq:    seq,
		mt:     HELLO,
		ack:    ack,
		buffer: pool.get(),
	}
	return wm.encode()
}

func newData(seq int32, data []byte, pool *pool) *wireMessage {
	buffer := pool.get()
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

func newAck(seqFor int32, pool *pool) *wireMessage {
	wm := &wireMessage{
		seq:    -1,
		mt:     ACK,
		ack:    seqFor,
		buffer: pool.get(),
	}
	return wm.encode()
}

func (self *wireMessage) rewriteAck(seqFor int32) {
	self.ack = seqFor
	WriteInt32(self.buffer.data[5:9], self.ack)
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

func mtString(mt messageType) string {
	if mt == HELLO {
		return "HELLO"
	} else if mt == ACK {
		return "ACK"
	} else if mt == DATA {
		return "DATA"
	} else if mt == CLOSE {
		return "CLOSE"
	} else {
		return "UNKNOWN"
	}
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