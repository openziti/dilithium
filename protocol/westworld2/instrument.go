package westworld2

import (
	"github.com/sirupsen/logrus"
	"net"
)

type Instrument interface {
	connected(peer *net.UDPAddr)
	wireMessageTx(peer *net.UDPAddr, wm *wireMessage)
	wireMessageRx(peer *net.UDPAddr, wm *wireMessage)
	wireMessageRetx(peer *net.UDPAddr, wm *wireMessage)
	unknownPeer(peer *net.UDPAddr)
	readError(peer *net.UDPAddr, err error)
	connectError(peer *net.UDPAddr, err error)
	unexpectedMessageType(peer *net.UDPAddr, mt messageType)
	duplicateRx(peer *net.UDPAddr, wm *wireMessage)
	duplicateAck(peer *net.UDPAddr, ack int32)
	allocate(ctx string)
}

type LoggerInstrument struct{}

func (self *LoggerInstrument) connected(peer *net.UDPAddr) {
	logrus.Infof("connected, peer [%s]", peer)
}

func (self *LoggerInstrument) wireMessageRx(peer *net.UDPAddr, wm *wireMessage) {
	logrus.Infof("<- [%c/#%d/@%d/:%d] <- [%s]", self.symbol(wm.mt), wm.seq, wm.ack, len(wm.data), peer)
}

func (self *LoggerInstrument) wireMessageTx(peer *net.UDPAddr, wm *wireMessage) {
	logrus.Infof("-> [%c/#%d/@%d/:%d] -> [%s]", self.symbol(wm.mt), wm.seq, wm.ack, len(wm.data), peer)
}

func (self *LoggerInstrument) wireMessageRetx(peer *net.UDPAddr, wm *wireMessage) {
	logrus.Warnf("!> [%c/#%d/@%d/:%d] -> [%s]", self.symbol(wm.mt), wm.seq, wm.ack, len(wm.data), peer)
}

func (self *LoggerInstrument) unknownPeer(peer *net.UDPAddr) {
	logrus.Errorf("unknown peer [%s]", peer)
}

func (self *LoggerInstrument) readError(peer *net.UDPAddr, err error) {
	logrus.Errorf("read error, peer [%s] (%v)", peer, err)
}

func (self *LoggerInstrument) connectError(peer *net.UDPAddr, err error) {
	logrus.Errorf("connect failed, peer [%s] (%v)", peer, err)
}

func (self *LoggerInstrument) unexpectedMessageType(peer *net.UDPAddr, mt messageType) {
	logrus.Errorf("unexpected message type [%s], peer [%s]", mtString(mt), peer)
}

func (self *LoggerInstrument) duplicateRx(peer *net.UDPAddr, wm *wireMessage) {
	logrus.Warnf("~ <- [#%d] <- [%s]", wm.seq, peer)
}

func (self *LoggerInstrument) duplicateAck(peer *net.UDPAddr, ack int32) {
	logrus.Warnf("~ <- [@%d] <- [%s]", ack, peer)
}

func (self *LoggerInstrument) allocate(ctx string) {
	logrus.WithField("context", ctx).Warn("allocate")
}

func (self *LoggerInstrument) symbol(mt messageType) rune {
	switch mt {
	case HELLO:
		return '&'
	case ACK:
		return '@'
	case DATA:
		return '#'
	case CLOSE:
		return '-'
	default:
		return '?'
	}
}
