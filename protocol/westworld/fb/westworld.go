package fb

import (
	flatbuffers "github.com/google/flatbuffers/go"
)

type WireModel struct {
	Sequence int32
	Type     Type
	Ack      int32
	Data     []byte
	buffer   []byte
	builder  *flatbuffers.Builder
}

func (self *WireModel) ToData() []byte {
	if self.buffer != nil {
		return self.buffer
	}
	return nil
}

func (self *WireModel) FromData(p []byte) *WireModel {
	wmin := GetRootAsWireMessage(p, 0)
	return &WireModel{
		Sequence: wmin.Sequence(),
		Type:     wmin.Type(),
		Ack:      wmin.Ack(),
	}
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
		builder:  builder,
	}
}
