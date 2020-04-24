package conduit

import (
	"bytes"
	"encoding/binary"
	"github.com/pkg/errors"
)

func newHelloMessage() *message {
	return &message{
		sequence: oobSequence,
		message:  Hello,
		payload:  nil,
	}
}

func newPayloadMessage(sequence int32, data []byte) *message {
	return &message{
		sequence: sequence,
		message:  Payload,
		payload:  data,
	}
}

type message struct {
	sequence int32
	message  messageType
	payload  []byte
}

func (self *message) marshal() ([]byte, error) {
	if self == nil {
		return nil, nil
	}

	data := new(bytes.Buffer)
	if _, err := data.Write(magicV1); err != nil {
		return nil, errors.Wrap(err, "magic write")
	}
	if err := binary.Write(data, binary.LittleEndian, self.sequence); err != nil {
		return nil, errors.Wrap(err, "sequence write")
	}
	if _, err := data.Write([]byte{uint8(self.message)}); err != nil {
		return nil, errors.Wrap(err, "message write")
	}
	if err := binary.Write(data, binary.LittleEndian, int32(len(self.payload))); err != nil {
		return nil, errors.Wrap(err, "payload length write")
	}
	if self.payload != nil {
		if n, err := data.Write(self.payload); err == nil {
			if n != len(self.payload) {
				return nil, errors.New("short payload write")
			}
		} else {
			return nil, errors.Wrap(err, "payload write")
		}
	}

	return data.Bytes(), nil
}

func unmarshal(data []byte) (*message, error) {
	if len(data) < headerLength {
		return nil, errors.New("short read")
	}

	m := &message{}
	for i := 0; i < len(magicV1); i++ {
		if data[i] != magicV1[i] {
			return nil, errors.New("bad magic")
		}
	}
	sequence, err := readInt32(data[2:6])
	if err != nil {
		return nil, errors.Wrap(err, "sequence read")
	}
	m.sequence = sequence
	m.message = messageType(data[6])
	payloadLength, err := readInt32(data[7:11])
	if err != nil {
		return nil, errors.Wrap(err, "payload length read")
	}
	m.payload = data[headerLength:]
	if int32(len(m.payload)) != payloadLength {
		return nil, errors.New("short payload")
	}

	return m, nil
}

var magicV1 = []byte{0x09, 0x09}

type messageType uint8

const oobSequence = -1
const headerLength = 2 + 4 + 1 + 4

const (
	Hello messageType = iota
	Payload
)
