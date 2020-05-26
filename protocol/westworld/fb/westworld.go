package fb

import (
	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/pkg/errors"
	"net"
)

func NewHello(sequence int32, builder *flatbuffers.Builder) []byte {
	builder.Reset()
	WireMessageStart(builder)
	WireMessageAddSequence(builder, sequence)
	WireMessageAddType(builder, TypeHELLO)
	builder.Finish(WireMessageEnd(builder))
	return builder.FinishedBytes()
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

func ReadWireMessage(conn *net.UDPConn) (wm *WireMessage, peer *net.UDPAddr, err error) {
	buffer := make([]byte, 64*1024)

	_, peer, err = conn.ReadFromUDP(buffer)
	if err != nil {
		return nil, peer, errors.Wrap(err, "read")
	}

	return GetRootAsWireMessage(buffer, 0), peer, nil
}
