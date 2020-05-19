package pb

import (
	"bytes"
	"encoding/binary"
	"github.com/golang/protobuf/proto"
	"github.com/michaelquigley/dilithium/util"
	"github.com/pkg/errors"
	"net"
)

func NewHello(seq int32) *WireMessage {
	return &WireMessage{
		Sequence: seq,
		Type:     MessageType_HELLO,
		Ack:      -1,
	}
}

func NewHelloAck(seq int32, ack int32) *WireMessage {
	return &WireMessage{
		Sequence: seq,
		Type:     MessageType_HELLO,
		Ack:      ack,
	}
}

func NewData(seq int32, p []byte) *WireMessage {
	wm := &WireMessage{
		Sequence: seq,
		Type:     MessageType_DATA,
		Ack:      -1,
		Data:     make([]byte, len(p)),
	}
	copy(wm.Data, p)
	return wm
}

func NewAck(forSequence int32) *WireMessage {
	return &WireMessage{
		Sequence: -1,
		Type:     MessageType_ACK,
		Ack:      forSequence,
	}
}

func ReadWireMessage(conn *net.UDPConn) (wm *WireMessage, peer *net.UDPAddr, err error) {
	buffer := make([]byte, 64*1024)

	var n int
	n, peer, err = conn.ReadFromUDP(buffer)
	if err != nil {
		return nil, peer, errors.Wrap(err, "read from peer")
	}

	wm, err = FromData(buffer[:n])
	if err != nil {
		return nil, peer, errors.Wrap(err, "from data")
	}

	return wm, peer, err
}

func WriteWireMessage(wm *WireMessage, conn *net.UDPConn, peer *net.UDPAddr) error {
	data, err := wm.ToData()
	if err != nil {
		return errors.Wrap(err, "to data")
	}

	n, err := conn.WriteToUDP(data, peer)
	if err != nil {
		return errors.Wrap(err, "write to peer")
	}
	if n != len(data) {
		return errors.New("short write to peer")
	}

	return nil
}

func (self *WireMessage) ToData() ([]byte, error) {
	data, err := proto.Marshal(self)
	if err != nil {
		return nil, errors.Wrap(err, "marshal")
	}

	buffer := new(bytes.Buffer)
	if err := binary.Write(buffer, binary.LittleEndian, int32(len(data))); err != nil {
		return nil, errors.Wrap(err, "length write")
	}
	n, err := buffer.Write(data)
	if err != nil {
		return nil, errors.Wrap(err, "data write")
	}
	if n != len(data) {
		return nil, errors.New("short data write")
	}

	return buffer.Bytes(), nil
}

func FromData(p []byte) (*WireMessage, error) {
	_, err := util.ReadInt32(p[:4])
	if err != nil {
		return nil, errors.Wrap(err, "length read")
	}

	wm := &WireMessage{}
	if err := proto.Unmarshal(p[4:], wm); err != nil {
		return nil, errors.Wrap(err, "unmarshal")
	}

	return wm, nil
}
