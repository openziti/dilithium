package wb

import (
	"encoding/binary"
	"github.com/michaelquigley/dilithium/util"
	"github.com/pkg/errors"
	"net"
	"sync"
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
	pool     *sync.Pool
}

func NewHello(sequence int32, buffer []byte, pool *sync.Pool) (*WireMessage, error) {
	wm := &WireMessage{
		Sequence: sequence,
		Type:     HELLO,
		Ack:      -1,
		Data:     nil,
		buffer:   buffer,
		pool:     pool,
	}
	return wm.encode()
}

func NewHelloAck(sequence, ack int32, buffer []byte, pool *sync.Pool) (*WireMessage, error) {
	wm := &WireMessage{
		Sequence: sequence,
		Type:     HELLO,
		Ack:      ack,
		Data:     nil,
		buffer:   buffer,
		pool:     pool,
	}
	return wm.encode()
}

func NewData(sequence int32, data []byte, buffer []byte, pool *sync.Pool) (*WireMessage, error) {
	wm := &WireMessage{
		Sequence: sequence,
		Type:     DATA,
		Ack:      -1,
		Data:     data,
		buffer:   buffer,
		pool:     pool,
	}
	return wm.encode()
}

func NewAck(forSequence int32, buffer []byte, pool *sync.Pool) (*WireMessage, error) {
	wm := &WireMessage{
		Sequence: -1,
		Type:     ACK,
		Ack:      forSequence,
		Data:     nil,
		buffer:   buffer,
		pool:     pool,
	}
	return wm.encode()
}

func (self *WireMessage) ToBuffer() []byte {
	return self.buffer[:self.len]
}

func (self *WireMessage) encode() (*WireMessage, error) {
	buffer := util.NewByteWriter(self.buffer)
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

func FromBuffer(buffer []byte, pool *sync.Pool) (*WireMessage, error) {
	wm := &WireMessage{buffer: buffer, pool: pool}
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

func ReadWireMessage(conn *net.UDPConn, buffer []byte, pool *sync.Pool) (wm *WireMessage, peer *net.UDPAddr, err error) {
	_, peer, err = conn.ReadFromUDP(buffer)
	if err == nil {
		return nil, peer, errors.Wrap(err, "read from peer")
	}

	wm, err = FromBuffer(buffer, pool)
	if err != nil {
		return nil, peer, errors.Wrap(err, "from data")
	}

	return wm, peer, err
}

func (self *WireMessage) WriteMessage(conn *net.UDPConn, peer *net.UDPAddr) error {
	n, err := conn.WriteToUDP(self.buffer[:self.len], peer)
	if err != nil {
		return errors.Wrap(err, "write to peer")
	}
	if n != self.len {
		return errors.New("short write to peer")
	}

	return nil
}