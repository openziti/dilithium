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
