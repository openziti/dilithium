package loop

import (
	"github.com/michaelquigley/dilithium/util"
	"github.com/pkg/errors"
)

type message struct {
	seq    uint32
	mt     messageType
	hash   []byte
	data   []byte
	buffer *buffer
}

func (self *message) encode() {
	hashLen := len(self.hash)
	util.WriteUint32(self.buffer.data[0:4], self.seq)
	self.buffer.data[4] = byte(self.mt)
	self.buffer.data[5] = byte(hashLen)
	for i := 0; i < hashLen; i++ {
		self.buffer.data[6+i] = self.hash[i]
	}
	for i := 0; i < len(self.data); i++ {
		self.buffer.data[6+hashLen+i] = self.data[i]
	}
}

func decode(buffer *buffer) (*message, error) {
	headerLen := buffer.data[5]
	if uint32(5+headerLen) > buffer.sz {
		return nil, errors.Errorf("short buffer [%d != %d]", 5+headerLen, buffer.sz)
	}
	m := &message{
		seq:    util.ReadUint32(buffer.data[0:4]),
		mt:     messageType(buffer.data[4]),
		hash:   buffer.data[6 : 6+headerLen],
		data:   buffer.data[6+headerLen:],
		buffer: buffer,
	}
	return m, nil
}

type messageType uint8

const (
	START messageType = iota
	DATA
	END
)

func (self messageType) string() string {
	if self == START {
		return "START"
	} else if self == DATA {
		return "DATA"
	} else if self == END {
		return "END"
	} else {
		panic("unknown message type")
	}
}
