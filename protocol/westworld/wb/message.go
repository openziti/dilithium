package wb

import (
	"bytes"
	"encoding/binary"
	"github.com/michaelquigley/dilithium/util"
	"github.com/pkg/errors"
)

type Type uint8

const (
	HELLO Type = iota
	DATA
	ACK
	CLOSE
)

type WireMessage struct {
	Sequence int32
	Type     Type
	Ack      int32
	Data     []byte
	buffer   []byte
	len      int
}

func NewHello(sequence int32, buffer []byte) (*WireMessage, error) {
	wm := &WireMessage{
		Sequence: sequence,
		Type:     HELLO,
		Ack:      -1,
		Data:     nil,
		buffer:   buffer,
	}
	return wm.encode()
}

func NewHelloAck(sequence, ack int32) (*WireMessage, error) {
	wm := &WireMessage{
		Sequence: sequence,
		Type:     HELLO,
		Ack:      ack,
		Data:     nil,
	}
	return wm.encode()
}

func (self *WireMessage) encode() (*WireMessage, error) {
	buffer := bytes.NewBuffer(self.buffer)
	if err := binary.Write(buffer, binary.LittleEndian, self.Sequence); err != nil {
		return nil, err
	}
	if err := binary.Write(buffer, binary.LittleEndian, uint8(self.Type)); err != nil {
		return nil, err
	}
	if err := binary.Write(buffer, binary.LittleEndian, self.Ack); err != nil {
		return nil, err
	}
	if err := binary.Write(buffer, binary.LittleEndian, int32(len(self.Data))); err != nil {
		return nil, err
	}
	if self.Data != nil && len(self.Data) > 0 {
		n, err := buffer.Write(self.Data)
		if err != nil {
			return nil, err
		}
		if n != len(self.Data) {
			return nil, errors.New("short write")
		}
	}
	self.Data = self.buffer[13 : 13+len(self.Data)]
	self.len = buffer.Len()
	return self, nil
}

func FromBuffer(buffer []byte) (*WireMessage, error) {
	wm := &WireMessage{buffer: buffer}
	if sequence, err := util.ReadInt32(buffer[:4]); err == nil {
		wm.Sequence = sequence
	} else {
		return nil, err
	}
	if type_, err := util.ReadUint8(buffer[4:5]); err == nil {
		wm.Type = Type(type_)
	} else {
		return nil, err
	}
	if ack, err := util.ReadInt32(buffer[5:9]); err == nil {
		wm.Ack = ack
	} else {
		return nil, err
	}
	if len, err := util.ReadInt32(buffer[9:13]); err == nil {
		wm.Data = buffer[13 : 13+len]
		wm.len = 13 + int(len)
	} else {
		return nil, err
	}
	return wm, nil
}
