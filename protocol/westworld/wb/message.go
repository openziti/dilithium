package wb

import (
	"encoding/binary"
	"github.com/michaelquigley/dilithium/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"net"
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
	pool     *BufferPool
}

func NewHello(sequence int32, pool *BufferPool) (*WireMessage, error) {
	wm := &WireMessage{
		Sequence: sequence,
		Type:     HELLO,
		Ack:      -1,
		Data:     nil,
		buffer:   pool.Get().([]byte),
		pool:     pool,
	}
	return wm.encode()
}

func NewHelloAck(sequence, ack int32, pool *BufferPool) (*WireMessage, error) {
	wm := &WireMessage{
		Sequence: sequence,
		Type:     HELLO,
		Ack:      ack,
		Data:     nil,
		buffer:   pool.Get().([]byte),
		pool:     pool,
	}
	return wm.encode()
}

func NewData(sequence int32, data []byte, pool *BufferPool) (*WireMessage, error) {
	wm := &WireMessage{
		Sequence: sequence,
		Type:     DATA,
		Ack:      -1,
		Data:     data,
		buffer:   pool.Get().([]byte),
		pool:     pool,
	}
	return wm.encode()
}

func NewAck(forSequence int32, pool *BufferPool) (*WireMessage, error) {
	wm := &WireMessage{
		Sequence: -1,
		Type:     ACK,
		Ack:      forSequence,
		Data:     nil,
		buffer:   pool.Get().([]byte),
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

func FromBuffer(buffer []byte, pool *BufferPool) (*WireMessage, error) {
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

func ReadWireMessage(conn *net.UDPConn, pool *BufferPool) (wm *WireMessage, peer *net.UDPAddr, err error) {
	buffer := pool.Get().([]byte)

	_, peer, err = conn.ReadFromUDP(buffer)
	if err != nil {
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

func (self *WireMessage) RewriteAck(ack int32) error {
	buffer := util.NewByteWriter(self.buffer[5:9])
	if err := binary.Write(buffer, binary.LittleEndian, self.Ack); err != nil {
		return err
	}
	self.Ack = ack
	return nil
}

func (self *WireMessage) Free() {
	if self.pool != nil && self.buffer != nil {
		self.pool.Put(self.buffer)
		self.pool = nil
		self.buffer = nil
	} else {
		logrus.Warnf("double-free")
	}
}
