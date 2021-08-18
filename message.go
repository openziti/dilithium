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
