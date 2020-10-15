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

type messageFlag uint8

const (
	// 0x8 ... 0x80
	RTT        messageFlag = 0x8
	INLINE_ACK messageFlag = 0x10
)

const headerSz = 7

func (self *wireMessage) encode() (*wireMessage, error) {
	dataSz := uint16(len(self.data))
	if self.buffer.sz < uint32(headerSz+dataSz) {
		return nil, errors.Errorf("short buffer for encode [%d < %d]", self.buffer.sz, headerSz+dataSz)
	}
	util.WriteInt32(self.buffer.data[0:4], self.seq)
	self.buffer.data[4] = byte(self.mt)
	util.WriteUint16(self.buffer.data[5:headerSz], dataSz)
	copy(self.buffer.data[headerSz:], self.data)
	self.buffer.uz = uint32(headerSz + len(self.data))
	return self, nil
}

func decode(buffer *buffer) (*wireMessage, error) {
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
