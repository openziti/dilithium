package westworld2

import (
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"net"
	"strings"
)

type wireMessage struct {
	seq    int32
	mt     messageType
	mf     messageFlag
	ack    int32
	data   []byte
	buffer *buffer
}

func readWireMessage(conn *net.UDPConn, pool *pool, i Instrument) (wm *wireMessage, peer *net.UDPAddr, err error) {
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

	if i != nil {
		i.wireMessageRx(peer, wm)
	}

	return
}

func writeWireMessage(wm *wireMessage, conn *net.UDPConn, peer *net.UDPAddr, i Instrument) error {
	if wm.buffer.sz < headerSz {
		logrus.Fatalf("truncated buffer!")
	}

	n, err := conn.WriteToUDP(wm.buffer.data[:wm.buffer.sz], peer)
	if err != nil {
		return errors.Wrap(err, "peer write")
	}
	if uint16(n) != wm.buffer.sz {
		return errors.Errorf("short peer write [%d != %d]", n, wm.buffer.sz)
	}
	if i != nil {
		i.wireMessageTx(peer, wm)
	}
	return nil
}

func (self *wireMessage) clone() (*wireMessage, error) {
	clone := self.buffer.clone()
	return decode(clone)
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
	WriteInt32(self.buffer.data[6:10], self.ack)
}

func newData(seq int32, data []byte, pool *pool) *wireMessage {
	buffer := pool.get()
	n := copy(buffer.data[headerSz:], data)
	wm := &wireMessage{
		seq:    seq,
		mt:     DATA,
		ack:    -1,
		data:   buffer.data[headerSz : headerSz+n],
		buffer: buffer,
	}
	return wm.encode()
}

func newClose(seq int32, pool *pool) *wireMessage {
	wm := &wireMessage{
		seq:    seq,
		mt:     CLOSE,
		ack:    -1,
		buffer: pool.get(),
	}
	return wm.encode()
}

func (self *wireMessage) writeRtt(ts int64) {
	WriteInt64(self.buffer.data[headerSz+len(self.data):headerSz+len(self.data)+8], ts)
	self.buffer.sz = uint16(headerSz + len(self.data) + 8)
	self.mf |= RTT
	self.buffer.data[5] = byte(self.mf)
}

func (self *wireMessage) readRtt() (ts int64, err error) {
	dataLen := ReadUint16(self.buffer.data[10:headerSz])
	if 11+dataLen+8 > self.buffer.sz {
		return 0, errors.Errorf("short buffer [%d > %d]", 11+dataLen+8, self.buffer.sz)
	}
	ts = ReadInt64(self.buffer.data[headerSz+len(self.data) : headerSz+len(self.data)+8])
	return ts, nil
}

func (self *wireMessage) encode() *wireMessage {
	WriteInt32(self.buffer.data[0:4], self.seq)
	self.buffer.data[4] = byte(self.mt)
	self.buffer.data[5] = byte(self.mf)
	WriteInt32(self.buffer.data[6:10], self.ack)
	WriteUint16(self.buffer.data[10:headerSz], uint16(len(self.data)))
	self.buffer.sz = uint16(headerSz + len(self.data))
	return self
}

func decode(buffer *buffer) (*wireMessage, error) {
	dataLen := ReadUint16(buffer.data[10:headerSz])
	if 11+dataLen > buffer.sz {
		return nil, errors.Errorf("short buffer [%d != %d]", 11+dataLen, buffer.sz)
	}
	wm := &wireMessage{
		seq:    ReadInt32(buffer.data[0:4]),
		mt:     messageType(buffer.data[4]),
		mf:     messageFlag(buffer.data[5]),
		ack:    ReadInt32(buffer.data[6:10]),
		data:   buffer.data[headerSz : headerSz+dataLen],
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

func (self messageType) string() string {
	if self == HELLO {
		return "HELLO"
	} else if self == ACK {
		return "ACK"
	} else if self == DATA {
		return "DATA"
	} else if self == CLOSE {
		return "CLOSE"
	} else {
		return "UNKNOWN"
	}
}

type messageFlag uint8

const (
	RTT messageFlag = 1
)

func (self messageFlag) string() string {
	out := ""
	if self|RTT == 1 {
		out += " RTT"
	}
	return strings.TrimSpace(out)
}

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

const (
	headerSz = 12
)
