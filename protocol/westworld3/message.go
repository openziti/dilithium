package westworld3

import (
	"github.com/openziti/dilithium/util"
	"github.com/pkg/errors"
	"net"
	"strings"
)

type wireMessage struct {
	seq    int32
	mt     messageType
	buffer *buffer
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

func readWireMessage(conn *net.UDPConn, pool *pool) (wm *wireMessage, peer *net.UDPAddr, err error) {
	buffer := pool.get()
	var n int
	n, peer, err = conn.ReadFromUDP(buffer.data)
	if err != nil {
		return nil, peer, errors.Wrap(err, "peer read")
	}
	buffer.uz = uint32(n)

	wm, err = decodeHeader(buffer)
	if err != nil {
		return nil, peer, errors.Wrap(err, "decode")
	}

	return
}

func writeWireMessage(wm *wireMessage, conn *net.UDPConn, peer *net.UDPAddr) error {
	if wm.buffer.uz < dataStart {
		return errors.New("truncated buffer")
	}

	n, err := conn.WriteToUDP(wm.buffer.data[:wm.buffer.uz], peer)
	if err != nil {
		return errors.Wrap(err, "peer write")
	}
	if uint32(n) != wm.buffer.uz {
		return errors.Errorf("short peer write [%d != %d]", n, wm.buffer.uz)
	}

	return nil
}

func newHello(seq int32, h hello, a *ack, p *pool) (wm *wireMessage, err error) {
	wm = &wireMessage{
		seq:    seq,
		mt:     HELLO,
		buffer: p.get(),
	}
	var acksSz uint32
	var helloSz uint32
	if a != nil {
		wm.setFlag(INLINE_ACK)
		acksSz, err = encodeAcks([]ack{*a}, wm.buffer.data[dataStart:])
		if err != nil {
			return nil, errors.Wrap(err, "error encoding hello ack")
		}
	}
	helloSz, err = encodeHello(h, wm.buffer.data[dataStart+acksSz:])
	return wm.encodeHeader(uint16(acksSz + helloSz))
}

func (self *wireMessage) asHello() (h hello, a []ack, err error) {
	if self.messageType() != HELLO {
		return hello{}, nil, errors.Errorf("unexpected message type [%d], expected HELLO", self.messageType())
	}
	i := uint32(0)
	if self.hasFlag(INLINE_ACK) {
		a, i, err = decodeAcks(self.buffer.data[dataStart:])
		if err != nil {
			return hello{}, nil, errors.Wrap(err, "error decoding acks")
		}
	}
	h, _, err = decodeHello(self.buffer.data[dataStart+i:])
	if err != nil {
		return hello{}, nil, errors.Wrap(err, "error decoding hello")
	}
	return
}

func newAck(acks []ack, rxPortalSz int32, rtt *uint16, p *pool) (wm *wireMessage, err error) {
	wm = &wireMessage{
		seq:    -1,
		mt:     ACK,
		buffer: p.get(),
	}
	rttSz := uint32(0)
	if rtt != nil {
		if wm.buffer.sz < dataStart+2 {
			return nil, errors.Errorf("short buffer for ack [%d < %d]", wm.buffer.sz, dataStart+2)
		}
		wm.setFlag(RTT)
		util.WriteUint16(wm.buffer.data[dataStart:], *rtt)
		rttSz = 2
	}
	acksSz := uint32(0)
	if len(acks) > 0 {
		acksSz, err = encodeAcks(acks, wm.buffer.data[dataStart+rttSz:])
		if err != nil {
			return nil, errors.Wrap(err, "error encoding acks")
		}
	}
	if dataStart+rttSz+acksSz > wm.buffer.sz {
		return nil, errors.Errorf("short buffer for ack [%d < %d]", wm.buffer.sz, dataStart+acksSz)
	}
	util.WriteInt32(wm.buffer.data[dataStart+rttSz+acksSz:], rxPortalSz)
	return wm.encodeHeader(uint16(rttSz + acksSz + 4))
}

func (self *wireMessage) asAck() (a []ack, rxPortalSz int32, rtt *uint16, err error) {
	if self.messageType() != ACK {
		return nil, 0, nil, errors.Errorf("unexpected message type [%d], expected ACK", self.messageType())
	}
	i := uint32(0)
	if self.hasFlag(RTT) {
		if self.buffer.uz < dataStart+2 {
			return nil, 0, nil, errors.Errorf("short buffer for ack decode [%d < %d]", self.buffer.uz, dataStart+2)
		}
		rtt = new(uint16)
		*rtt = util.ReadUint16(self.buffer.data[dataStart:])
		i += 2
	}
	var acksSz uint32
	a, acksSz, err = decodeAcks(self.buffer.data[dataStart+i:])
	if err != nil {
		return nil, 0, nil, errors.Wrap(err, "error decoding acks")
	}
	i += acksSz
	if self.buffer.uz < i+4 {
		return nil, 0, nil, errors.Errorf("short buffer for rxPortalSz decode [%d < %d]", self.buffer.uz, i+4)
	}
	rxPortalSz = util.ReadInt32(self.buffer.data[dataStart+i:])
	return
}

func newData(seq int32, rtt *uint16, data []byte, p *pool) (wm *wireMessage, err error) {
	dataSz := uint32(len(data))
	wm = &wireMessage{
		seq:    seq,
		mt:     DATA,
		buffer: p.get(),
	}
	rttSz := uint32(0)
	if rtt != nil {
		if wm.buffer.sz < dataStart+2 {
			return nil, errors.Errorf("short buffer for rtt [%d < %d]", wm.buffer.sz, wm.buffer.sz+dataStart+2)
		}
		wm.setFlag(RTT)
		util.WriteUint16(wm.buffer.data[dataStart:], *rtt)
		rttSz = 2
	}
	if wm.buffer.sz < dataStart+rttSz+dataSz {
		return nil, errors.Errorf("short buffer for data [%d < %d]", wm.buffer.sz, dataStart+rttSz+dataSz)
	}
	copy(wm.buffer.data[dataStart+rttSz:], data)
	return wm.encodeHeader(uint16(rttSz+dataSz))
}

func (self *wireMessage) asData() (data []byte, rtt *uint16, err error) {
	if self.messageType() != DATA {
		return nil, nil, errors.Errorf("unexpected message type [%d], expected DATA", self.messageType())
	}
	rttSz := uint32(0)
	if self.hasFlag(RTT) {
		if self.buffer.uz < dataStart+2 {
			return nil, nil, errors.Errorf("short buffer for data decode [%d < %d]", self.buffer.uz, dataStart+2)
		}
		rtt = new(uint16)
		*rtt = util.ReadUint16(self.buffer.data[dataStart:])
		rttSz = 2
	}
	return self.buffer.data[dataStart+rttSz:self.buffer.uz], rtt, nil
}

func (self *wireMessage) asDataSize() (sz uint32, err error) {
	if self.messageType() != DATA {
		return 0, errors.Errorf("unexpected message type [%d], expected DATA", self.messageType())
	}
	rttSz := uint32(0)
	if self.hasFlag(RTT) {
		rttSz = 2
	}
	return self.buffer.uz - (dataStart + rttSz), nil
}

func newKeepalive(p *pool) (wm *wireMessage, err error) {
	return (&wireMessage{seq: -1, mt: KEEPALIVE, buffer: p.get()}).encodeHeader(0)
}

func newClose(seq int32, p *pool) (wm *wireMessage, err error) {
	return (&wireMessage{seq: seq, mt: CLOSE, buffer: p.get()}).encodeHeader(0)
}

func (self *wireMessage) encodeHeader(dataSz uint16) (*wireMessage, error) {
	if self.buffer.sz < uint32(dataStart+dataSz) {
		return nil, errors.Errorf("short buffer for encode [%d < %d]", self.buffer.sz, dataStart+dataSz)
	}
	util.WriteInt32(self.buffer.data[0:4], self.seq)
	self.buffer.data[4] = byte(self.mt)
	util.WriteUint16(self.buffer.data[5:dataStart], dataSz)
	self.buffer.uz = uint32(dataStart + dataSz)
	return self, nil
}

func decodeHeader(buffer *buffer) (*wireMessage, error) {
	sz := util.ReadUint16(buffer.data[5:dataStart])
	if uint32(dataStart+sz) > buffer.uz {
		return nil, errors.Errorf("short buffer read [%d != %d]", buffer.sz, dataStart+sz)
	}
	wm := &wireMessage{
		seq:    util.ReadInt32(buffer.data[0:4]),
		mt:     messageType(buffer.data[4]),
		buffer: buffer,
	}
	return wm, nil
}

func (self *wireMessage) insertData(data []byte) error {
	dataSz := uint16(len(data))
	if self.buffer.sz < self.buffer.uz+uint32(dataSz) {
		return errors.Errorf("short buffer for insert [%d < %d]", self.buffer.sz, self.buffer.uz+uint32(dataSz))
	}
	for i := self.buffer.uz - 1; i >= dataStart; i-- {
		self.buffer.data[i+uint32(dataSz)] = self.buffer.data[i]
	}
	for i := 0; i < int(dataSz); i++ {
		self.buffer.data[dataStart+i] = data[i]
	}
	self.buffer.uz = self.buffer.uz + uint32(dataSz)
	return nil
}

func (self *wireMessage) appendData(data []byte) error {
	dataSz := uint16(len(data))
	if self.buffer.sz < self.buffer.uz+uint32(dataSz) {
		return errors.Errorf("short buffer for append [%d < %d]", self.buffer.sz, self.buffer.uz+uint32(dataSz))
	}
	for i := 0; i < int(dataSz); i++ {
		self.buffer.data[self.buffer.uz+uint32(i)] = data[i]
	}
	self.buffer.uz = self.buffer.uz + uint32(dataSz)
	return nil
}

func (self *wireMessage) messageType() messageType {
	return messageType(byte(self.mt) & messageTypeMask)
}

func (self *wireMessage) setFlag(flag messageFlag) {
	self.mt = messageType(uint8(self.mt) | uint8(flag))
}

func (self *wireMessage) clearFlag(flag messageFlag) {
	self.mt = messageType(uint8(self.mt) ^ uint8(flag))
}

func (self *wireMessage) hasFlag(flag messageFlag) bool {
	if uint8(self.mt)&uint8(flag) > 0 {
		return true
	}
	return false
}

func (mt messageType) String() string {
	switch mt {
	case HELLO:
		return "HELLO"
	case ACK:
		return "ACK"
	case DATA:
		return "DATA"
	case KEEPALIVE:
		return "KEEPALIVE"
	case CLOSE:
		return "CLOSE"
	default:
		return "???"
	}
}

func (mt messageType) FlagsString() string {
	flags := ""
	if messageFlag(mt) & INLINE_ACK == INLINE_ACK {
		flags += " INLINE_ACK"
	}
	if messageFlag(mt) & RTT == RTT {
		flags += " RTT"
	}
	return strings.TrimSpace(flags)
}