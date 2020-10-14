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
	CLOSE
)

type messageFlag uint8

const (
	// 0x8 ... 0x80
	RTT messageFlag = 0x8
)

const headerSz = 7

func (self *wireMessage) encode() *wireMessage {
	util.WriteInt32(self.buffer.data[0:4], self.seq)
	self.buffer.data[4] = byte(self.mt)
	util.WriteUint16(self.buffer.data[5:headerSz], uint16(len(self.data)))
	self.buffer.sz = uint32(headerSz + len(self.data))
	return self
}

func decode(buffer *buffer) (*wireMessage, error) {
	sz := util.ReadUint16(buffer.data[5:headerSz])
	if uint32(headerSz+sz) > buffer.sz {
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
