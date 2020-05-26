package fb

import (
	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/pkg/errors"
	"net"
)

type WireModel struct {
	Sequence int32
	Type     Type
	Ack      int32
	Data     []byte
	out      []byte
	builder  *flatbuffers.Builder
}

func NewHello(sequence int32, builder *flatbuffers.Builder) *WireModel {
	builder.Reset()
	WireMessageStart(builder)
	WireMessageAddSequence(builder, sequence)
	WireMessageAddType(builder, TypeHELLO)
	WireMessageAddAck(builder, -1)
	builder.Finish(WireMessageEnd(builder))
	return &WireModel{
		Sequence: sequence,
		Type:     TypeHELLO,
		Ack:      -1,
		Data:     nil,
		out:      builder.FinishedBytes(),
		builder:  builder,
	}
}

func NewHelloAck(sequence int32, ack int32, builder *flatbuffers.Builder) []byte {
	builder.Reset()
	WireMessageStart(builder)
	WireMessageAddSequence(builder, sequence)
	WireMessageAddType(builder, TypeHELLO)
	WireMessageAddAck(builder, ack)
	builder.Finish(WireMessageEnd(builder))
	return builder.FinishedBytes()
}

func NewData(sequence int32, p []byte, builder *flatbuffers.Builder) []byte {
	builder.Reset()
	WireMessageStart(builder)
	WireMessageAddSequence(builder, sequence)
	WireMessageAddType(builder, TypeDATA)
	data := builder.CreateByteVector(p)
	WireMessageAddData(builder, data)
	builder.Finish(WireMessageEnd(builder))
	return builder.FinishedBytes()
}

func NewAck(forSequence int32, builder *flatbuffers.Builder) []byte {
	builder.Reset()
	WireMessageStart(builder)
	WireMessageAddType(builder, TypeACK)
	WireMessageAddAck(builder, forSequence)
	builder.Finish(WireMessageEnd(builder))
	return builder.FinishedBytes()
}

func ReadWireMessage(conn *net.UDPConn) (wm *WireModel, peer *net.UDPAddr, err error) {
	buffer := make([]byte, 64*1024)

	_, peer, err = conn.ReadFromUDP(buffer)
	if err != nil {
		return nil, peer, errors.Wrap(err, "read")
	}

	wmm := GetRootAsWireMessage(buffer, 0)

	return &WireModel{
		Sequence: wmm.Sequence(),
		Type: wmm.Type(),
		Ack: wmm.Ack(),
		// Data: ?
	}, peer, nil
}
