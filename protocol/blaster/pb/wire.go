package pb

import (
	"bytes"
	"encoding/binary"
	"github.com/michaelquigley/dilithium/util"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
	"io"
	"net"
)

type AddressedWireMessage struct {
	FromPeer    *net.UDPAddr
	WireMessage *WireMessage
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

func NewData(sequence int32, p []byte) *WireMessage {
	buffer := make([]byte, len(p))
	copy(buffer, p)
	return &WireMessage{
		Sequence: sequence,
		Type:     MessageType_DATA,
		DataPayload: &WireMessage_DataPayload{
			Data: buffer,
		},
	}
}

func NewEow(sequence int32, highWater int32) *WireMessage {
	return &WireMessage{
		Sequence: sequence,
		Type:     MessageType_EOW,
		EowPayload: &WireMessage_EowPayload{
			HighWater: highWater,
		},
	}
}

func ToData(wireMessage *WireMessage) ([]byte, error) {
	pbData, err := proto.Marshal(wireMessage)
	if err != nil {
		return nil, errors.Wrap(err, "marshal")
	}

	buffer := new(bytes.Buffer)
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

	return buffer.Bytes(), nil
}

func FromData(p []byte) (*WireMessage, error) {
	_, err := util.ReadInt32(p[:4])
	if err != nil {
		return nil, errors.Wrap(err, "read length")
	}
	wireMessage := &WireMessage{}
	if err := proto.Unmarshal(p[4:], wireMessage); err != nil {
		return nil, errors.Wrap(err, "unmarshal")
	}

	return wireMessage, nil
}

func WriteMessage(wireMessage *WireMessage, wr io.Writer) error {
	pbData, err := proto.Marshal(wireMessage)
	if err != nil {
		return errors.Wrap(err, "encode")
	}

	buffer := new(bytes.Buffer)
	if err := binary.Write(buffer, binary.LittleEndian, int32(len(pbData))); err != nil {
		return errors.Wrap(err, "length encode")
	}
	tn := buffer.Len()

	n, err := wr.Write(buffer.Bytes())
	if err != nil {
		return errors.Wrap(err, "length write")
	}
	if n != tn {
		return errors.Errorf("length short write [%d]", n)
	}

	n, err = wr.Write(pbData)
	if err != nil {
		return errors.Wrap(err, "write")
	}
	if n != len(pbData) {
		return errors.New("short write")
	}

	return nil
}

func ReadMessage(rd io.Reader) (*WireMessage, error) {
	lengthData := make([]byte, 4)
	n, err := io.ReadFull(rd, lengthData)
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
	n, err = io.ReadFull(rd, messageData)
	if err != nil {
		return nil, errors.Wrap(err, "message read")
	}
	if n != int(length) {
		return nil, errors.New("short message read")
	}

	wireMessage := &WireMessage{}
	if err := proto.Unmarshal(messageData, wireMessage); err != nil {
		return nil, errors.Wrap(err, "message unmarshal")
	}

	return wireMessage, nil
}
