package westworld3

import (
	"github.com/michaelquigley/dilithium/util"
	"github.com/pkg/errors"
)

type wireMessage struct {
	seq    int32
	mt     messageType
	data   []byte
	buffer *buffer
}

type messageType uint8

const (
	// 0x0 ... 0x7
	HELLO messageType = iota
	ACK
	DATA
	KEEPALIVE
	CLOSE
)

const messageTypeMask = byte(0x7)

type messageFlag uint8

const (
	// 0x8 ... 0x80
	RTT        messageFlag = 0x8
	INLINE_ACK messageFlag = 0x10
)

const headerSz = 7

func newHello(seq int32, h hello, a *ack, p *pool) (wm *wireMessage, err error) {
	wm = &wireMessage{
		seq:    seq,
		mt:     HELLO,
		buffer: p.get(),
	}
	var acksSz uint32
	var helloSz uint32
	if a != nil {
		wm.setFlag(INLINE_ACK)
		acksSz, err = encodeAcks([]ack{*a}, wm.buffer.data[headerSz:])
		if err != nil {
			return nil, errors.Wrap(err, "error encoding hello ack")
		}
	}
	helloSz, err = encodeHello(h, wm.buffer.data[headerSz+acksSz:])
	return wm.encodeHeader(uint16(acksSz + helloSz))
}

func (self *wireMessage) asHello() (h hello, a []ack, err error) {
	if self.messageType() != HELLO {
		return hello{}, nil, errors.Errorf("unexpected message type [%d], expected HELLO", self.messageType())
	}
	i := uint32(0)
	if self.hasFlag(INLINE_ACK) {
		a, i, err = decodeAcks(self.data)
		if err != nil {
			return hello{}, nil, errors.Wrap(err, "error decoding acks")
		}
	}
	h, _, err = decodeHello(self.data[i:])
	if err != nil {
		return hello{}, nil, errors.Wrap(err, "error decoding hello")
	}
	return
}

func newAck(acks []ack, rxPortalSz int, pool *pool) (wm *wireMessage, err error) {
	wm = &wireMessage{
		seq: -1,
		mt:  ACK,
	}
	var acksSz uint32
	if len(acks) > 0 {
		acksSz, err = encodeAcks(acks, wm.buffer.data[headerSz:])
		if err != nil {
			return nil, errors.Wrap(err, "error encoding acks")
		}
	}
	if headerSz+acksSz > wm.buffer.sz {
		return nil, errors.Errorf("short buffer for ack [%d < %d]", wm.buffer.sz, headerSz+acksSz)
	}
	util.WriteInt32(wm.buffer.data[headerSz+acksSz:], int32(rxPortalSz))
	return wm.encodeHeader(uint16(acksSz + 4))
}

func (self *wireMessage) encodeHeader(dataSz uint16) (*wireMessage, error) {
	if self.buffer.sz < uint32(headerSz+dataSz) {
		return nil, errors.Errorf("short buffer for encode [%d < %d]", self.buffer.sz, headerSz+dataSz)
	}
	util.WriteInt32(self.buffer.data[0:4], self.seq)
	self.buffer.data[4] = byte(self.mt)
	util.WriteUint16(self.buffer.data[5:headerSz], dataSz)
	self.buffer.uz = uint32(headerSz + dataSz)
	return self, nil
}

func decodeHeader(buffer *buffer) (*wireMessage, error) {
	sz := util.ReadUint16(buffer.data[5:headerSz])
	if uint32(headerSz+sz) > buffer.uz {
		return nil, errors.Errorf("short buffer read [%d != %d]", buffer.sz, headerSz+sz)
	}
	wm := &wireMessage{
		seq:    util.ReadInt32(buffer.data[0:4]),
		mt:     messageType(buffer.data[4]),
		data:   buffer.data[headerSz : headerSz+sz],
		buffer: buffer,
	}
	return wm, nil
}

func (self *wireMessage) insertData(data []byte) error {
	dataSz := uint16(len(data))
	if self.buffer.sz < self.buffer.uz+uint32(dataSz) {
		return errors.Errorf("short buffer for insert [%d < %d]", self.buffer.sz, self.buffer.uz+uint32(dataSz))
	}
	for i := self.buffer.uz - 1; i >= headerSz; i-- {
		self.buffer.data[i+uint32(dataSz)] = self.buffer.data[i]
	}
	for i := 0; i < int(dataSz); i++ {
		self.buffer.data[headerSz+i] = data[i]
	}
	self.buffer.uz = self.buffer.uz + uint32(dataSz)
	return nil
}

func (self *wireMessage) appendData(data []byte) error {
	dataSz := uint16(len(data))
	if self.buffer.sz < self.buffer.uz+uint32(dataSz) {
		return errors.Errorf("short buffer for append [%d < %d]", self.buffer.sz, self.buffer.uz+uint32(dataSz))
	}
	for i := 0; i < int(dataSz); i++ {
		self.buffer.data[self.buffer.uz+uint32(i)] = data[i]
	}
	self.buffer.uz = self.buffer.uz + uint32(dataSz)
	return nil
}

func (self *wireMessage) messageType() messageType {
	return messageType(byte(self.mt) & messageTypeMask)
}

func (self *wireMessage) setFlag(flag messageFlag) {
	self.mt = messageType(uint8(self.mt) | uint8(flag))
}

func (self *wireMessage) clearFlag(flag messageFlag) {
	self.mt = messageType(uint8(self.mt) ^ uint8(flag))
}

func (self *wireMessage) hasFlag(flag messageFlag) bool {
	if uint8(self.mt)&uint8(flag) > 0 {
		return true
	}
	return false
}
