package pb

import (
	"bytes"
	"encoding/binary"
	"github.com/michaelquigley/dilithium/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
	"io"
	"net"
)

type WireMessagePeer struct {
	WireMessage *WireMessage
	Peer        *net.UDPAddr
}

func NewSync(sequence int32) *WireMessage {
	return &WireMessage{
		Sequence: sequence,
		Type:     MessageType_SYNC,
	}
}

func NewHello(sequence int32, session string) *WireMessage {
	return &WireMessage{
		Sequence: sequence,
		Type:     MessageType_HELLO,
		HelloPayload: &WireMessage_HelloPayload{
			Session: session,
		},
	}
}

func NewOk(sequence int32) *WireMessage {
	return &WireMessage{
		Sequence: sequence,
		Type:     MessageType_OK,
	}
}

func ToData(wireMessage *WireMessage) ([]byte, error) {
	pbData, err := proto.Marshal(wireMessage)
	if err != nil {
		return nil, errors.Wrap(err, "marshal")
	}

	data := make([]byte, len(pbData)+4)
	buffer := bytes.NewBuffer(data)
	if err := binary.Write(buffer, binary.LittleEndian, int32(len(pbData))); err != nil {
		return nil, errors.Wrap(err, "write length")
	}
	n, err := buffer.Write(pbData)
	if err != nil {
		return nil, errors.Wrap(err, "write pb data")
	}
	if n != len(pbData) {
		return nil, errors.New("short pb data")
	}

	return data, nil
}

func FromData(p []byte) (*WireMessage, error) {
	pbLen, err := util.ReadInt32(p[:4])
	if err != nil {
		return nil, errors.Wrap(err, "read length")
	}
	if len(p) != int(pbLen+4) {
		return nil, errors.New("buffer length")
	}

	wireMessage := &WireMessage{}
	if err := proto.Unmarshal(p[4:], wireMessage); err != nil {
		return nil, errors.Wrap(err, "unmarshal")
	}

	return wireMessage, nil
}

func ReadMessage(conn net.Conn) (*WireMessage, error) {
	lengthData := make([]byte, 4)
	n, err := io.ReadFull(conn, lengthData)
	if err != nil {
		return nil, errors.Wrap(err, "length read")
	}
	if n != 4 {
		return nil, errors.New("short length read")
	}
	length, err := util.ReadInt32(lengthData)
	if err != nil {
		return nil, errors.Wrap(err, "length unmarshal")
	}

	messageData := make([]byte, length)
	n, err = io.ReadFull(conn, messageData)
	if err != nil {
		return nil, errors.Wrap(err, "message read")
	}
	if n != int(length) {
		return nil, errors.New("short message read")
	}

	logrus.Infof("read [%d] message bytes", length)

	wireMessage := &WireMessage{}
	if err := proto.Unmarshal(messageData, wireMessage); err != nil {
		return nil, errors.Wrap(err, "message unmarshal")
	}

	return wireMessage, nil
}

func WriteMessage(wireMessage *WireMessage, conn net.Conn) error {
	data, err := ToData(wireMessage)
	if err != nil {
		return errors.Wrap(err, "encode")
	}
	n, err := conn.Write(data)
	if err != nil {
		return errors.Wrap(err, "write")
	}
	if n != len(data) {
		return errors.New("short write")
	}
	return nil
}