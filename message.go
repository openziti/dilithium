package dilithium

import (
	"github.com/openziti/dilithium/util"
	"github.com/pkg/errors"
)

type WireMessage struct {
	Seq int32
	Mt  messageType
	buf *Buffer
}

type messageType uint8

const (
	// 0x0 ... 0x7
	HELLO messageType = iota
	ACK
	DATA
	KEEPALIVE
	CLOSE
)

const messageTypeMask = byte(0x7)

type messageFlag uint8

const (
	// 0x8 ... 0x80
	RTT        messageFlag = 0x8
	INLINE_ACK messageFlag = 0x10
)

const dataStart = 7

func readWireMessage(t Transport, pool *Pool) (wm *WireMessage, err error) {
	buf := pool.Get()
	var n int
	n, err = t.Read(buf.Data)
	if err != nil {
		return nil, errors.Wrap(err, "peer read")
	}
	buf.Used = uint32(n)

	wm, err = decodeHeader(buf)
	if err != nil {
		return nil, errors.Wrap(err, "decode header")
	}

	return
}

func writeWireMessage(wm *WireMessage, t Transport) error {
	if wm.buf.Used < dataStart {
		return errors.New("truncated buffer")
	}

	n, err := t.Write(wm.buf.Data[:wm.buf.Used])
	if err != nil {
		return errors.Wrap(err, "write")
	}
	if uint32(n) != wm.buf.Used {
		return errors.Errorf("short write [%d != %d]", n, wm.buf.Used)
	}

	return nil
}

func newHello(seq int32, h hello, a *Ack, p *Pool) (wm *WireMessage, err error) {
	wm = &WireMessage{
		Seq: seq,
		Mt:  HELLO,
		buf: p.Get(),
	}
	var ackSize uint32
	var helloSize uint32
	if a != nil {
		wm.setFlag(INLINE_ACK)
		ackSize, err = EncodeAcks([]Ack{*a}, wm.buf.Data[dataStart:])
		if err != nil {
			return nil, errors.Wrap(err, "error encoding hello ack")
		}
	}
	helloSize, err = encodeHello(h, wm.buf.Data[dataStart+ackSize:])
	if err != nil {
		return nil, errors.Wrap(err, "error encoding hello")
	}
	return wm.encodeHeader(uint16(ackSize + helloSize))
}

func (wm *WireMessage) asHello() (h hello, a []Ack, err error) {
	if wm.messageType() != HELLO {
		return hello{}, nil, errors.Errorf("unexpected message type [%d], expected HELLO", wm.messageType())
	}
	i := uint32(0)
	if wm.hasFlag(INLINE_ACK) {
		a, i, err = DecodeAcks(wm.buf.Data[dataStart:])
		if err != nil {
			return hello{}, nil, errors.Wrap(err, "error decoding acks")
		}
	}
	h, _, err = decodeHello(wm.buf.Data[dataStart+i:])
	if err != nil {
		return hello{}, nil, errors.Wrap(err, "error decoding hello")
	}
	return
}

func newAck(acks []Ack, rxPortalSize int32, rtt *uint16, pool *Pool) (wm *WireMessage, err error) {
	wm = &WireMessage{
		Seq: -1,
		Mt:  ACK,
		buf: pool.Get(),
	}
	rttSize := uint32(0)
	if rtt != nil {
		if wm.buf.Size < dataStart+2 {
			return nil, errors.Errorf("short buffer for ack [%d < %d]", wm.buf.Size, dataStart+2)
		}
		wm.setFlag(RTT)
		util.WriteUint16(wm.buf.Data[dataStart:], *rtt)
		rttSize = 2
	}
	ackSize := uint32(0)
	if len(acks) > 0 {
		ackSize, err = EncodeAcks(acks, wm.buf.Data[dataStart+rttSize:])
		if err != nil {
			return nil, errors.Wrap(err, "error encoding acks")
		}
	}
	if dataStart+rttSize+ackSize > wm.buf.Size {
		return nil, errors.Errorf("short buffer for ack [%d < %d]", wm.buf.Size, dataStart+rttSize+ackSize)
	}
	util.WriteInt32(wm.buf.Data[dataStart+rttSize+ackSize:], rxPortalSize)
	return wm.encodeHeader(uint16(rttSize + ackSize + 4))
}

func (wm *WireMessage) asAck() (a []Ack, rxPortalSize int32, rtt *uint16, err error) {
	if wm.messageType() != ACK {
		return nil, 0, nil, errors.Errorf("unexpected message type [%d], expected ACK", wm.messageType())
	}
	i := uint32(0)
	if wm.hasFlag(RTT) {
		if wm.buf.Used < dataStart+2 {
			return nil, 0, nil, errors.Errorf("short buffer for ack decode [%d < %d]", wm.buf.Used, dataStart+2)
		}
		rtt = new(uint16)
		*rtt = util.ReadUint16(wm.buf.Data[dataStart:])
		i += 2
	}
	var ackSize uint32
	a, ackSize, err = DecodeAcks(wm.buf.Data[dataStart+i:])
	if err != nil {
		return nil, 0, nil, errors.Wrap(err, "error decoding acks")
	}
	i += ackSize
	if wm.buf.Used < i+4 {
		return nil, 0, nil, errors.Errorf("short buffer for rxPortalSize decode [%d < %d]", wm.buf.Used, i+4)
	}
	rxPortalSize = util.ReadInt32(wm.buf.Data[dataStart+i:])
	return
}

func newData(seq int32, rtt *uint16, data []byte, pool *Pool) (wm *WireMessage, err error) {
	dataSize := uint32(len(data))
	wm = &WireMessage{
		Seq: seq,
		Mt:  DATA,
		buf: pool.Get(),
	}
	rttSize := uint32(0)
	if rtt != nil {
		if wm.buf.Size < dataStart+2 {
			return nil, errors.Errorf("short buffer for rtt [%d < %d]", wm.buf.Size, dataStart+2)
		}
		wm.setFlag(RTT)
		util.WriteUint16(wm.buf.Data[dataStart:], *rtt)
		rttSize = 2
	}
	if wm.buf.Size < dataStart+rttSize+dataSize {
		return nil, errors.Errorf("short buffer for data [%d < %d]", wm.buf.Size, dataStart+rttSize+dataSize)
	}
	copy(wm.buf.Data[dataStart+rttSize:], data)
	return wm.encodeHeader(uint16(rttSize + dataSize))
}

func (wm *WireMessage) asData() (data []byte, rtt *uint16, err error) {
	if wm.messageType() != DATA {
		return nil, nil, errors.Errorf("unexpected message type [%d], expected DATA", wm.messageType())
	}
	rttSize := uint32(0)
	if wm.hasFlag(RTT) {
		if wm.buf.Used < dataStart+2 {
			return nil, nil, errors.Errorf("short buffer for data decode [%d < %d]", wm.buf.Used, dataStart+2)
		}
		rtt = new(uint16)
		*rtt = util.ReadUint16(wm.buf.Data[dataStart:])
		rttSize = 2
	}
	return wm.buf.Data[dataStart+rttSize : wm.buf.Used], rtt, nil
}

func (wm *WireMessage) asDataSize() (size uint32, err error) {
	if wm.messageType() != DATA {
		return 0, errors.Errorf("unexpected message type [%d], expected DATA", wm.messageType())
	}
	rttSize := uint32(0)
	if wm.hasFlag(RTT) {
		rttSize = 2
	}
	return wm.buf.Used - (dataStart + rttSize), nil
}

func (wm *WireMessage) encodeHeader(dataSize uint16) (*WireMessage, error) {
	if wm.buf.Size < uint32(dataStart+dataSize) {
		return nil, errors.Errorf("short buffer for encode [%d < %d]", wm.buf.Size, dataStart+dataSize)
	}
	util.WriteInt32(wm.buf.Data[0:4], wm.Seq)
	wm.buf.Data[4] = byte(wm.Mt)
	util.WriteUint16(wm.buf.Data[5:dataStart], dataSize)
	wm.buf.Used = uint32(dataStart + dataSize)
	return wm, nil
}

func decodeHeader(buf *Buffer) (*WireMessage, error) {
	size := util.ReadUint16(buf.Data[5:dataStart])
	if uint32(dataStart+size) > buf.Used {
		return nil, errors.Errorf("short buffer read [%d != %d]", buf.Size, dataStart+size)
	}
	wm := &WireMessage{
		Seq: util.ReadInt32(buf.Data[0:4]),
		Mt:  messageType(buf.Data[4]),
		buf: buf,
	}
	return wm, nil
}

func (wm *WireMessage) messageType() messageType {
	return messageType(byte(wm.Mt) & messageTypeMask)
}

func (wm *WireMessage) setFlag(flag messageFlag) {
	wm.Mt = messageType(uint8(wm.Mt) | uint8(flag))
}

func (wm *WireMessage) hasFlag(flag messageFlag) bool {
	if uint8(wm.Mt)&uint8(flag) > 0 {
		return true
	}
	return false
}

func (wm *WireMessage) clearFlag(flag messageFlag) {
	wm.Mt = messageType(uint8(wm.Mt) ^ uint8(flag))
}
