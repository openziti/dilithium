package westworld2

import (
	"github.com/sirupsen/logrus"
	"net"
)

type loggerInstrument struct{}

func NewLoggerInstrument() Instrument {
	return &loggerInstrument{}
}

func (self *loggerInstrument) connected(peer *net.UDPAddr) {
	logrus.Infof("connected, peer [%s]", peer)
}

func (self *loggerInstrument) wireMessageRx(peer *net.UDPAddr, wm *wireMessage) {
	logrus.Infof("<- [%c/#%d/@%d/:%d] <- [%s]", self.symbol(wm.mt), wm.seq, wm.ack, len(wm.data), peer)
}

func (self *loggerInstrument) wireMessageTx(peer *net.UDPAddr, wm *wireMessage) {
	logrus.Infof("-> [%c/#%d/@%d/:%d] -> [%s]", self.symbol(wm.mt), wm.seq, wm.ack, len(wm.data), peer)
}

func (self *loggerInstrument) wireMessageRetx(peer *net.UDPAddr, wm *wireMessage) {
	logrus.Warnf("!> [%c/#%d/@%d/:%d] -> [%s]", self.symbol(wm.mt), wm.seq, wm.ack, len(wm.data), peer)
}

func (self *loggerInstrument) unknownPeer(peer *net.UDPAddr) {
	logrus.Errorf("unknown peer [%s]", peer)
}

func (self *loggerInstrument) readError(peer *net.UDPAddr, err error) {
	logrus.Errorf("read error, peer [%s] (%v)", peer, err)
}

func (self *loggerInstrument) connectError(peer *net.UDPAddr, err error) {
	logrus.Errorf("connect failed, peer [%s] (%v)", peer, err)
}

func (self *loggerInstrument) unexpectedMessageType(peer *net.UDPAddr, mt messageType) {
	logrus.Errorf("unexpected message type [%s], peer [%s]", mt.string(), peer)
}

func (self *loggerInstrument) duplicateRx(peer *net.UDPAddr, wm *wireMessage) {
	logrus.Warnf("~ <- [#%d] <- [%s]", wm.seq, peer)
}

func (self *loggerInstrument) duplicateAck(peer *net.UDPAddr, ack int32) {
	logrus.Warnf("~ <- [@%d] <- [%s]", ack, peer)
}

func (self *loggerInstrument) newRextMs(peer *net.UDPAddr, rextMs int) {
	logrus.Infof("!+[%d ms]  <- [%s]", rextMs, peer)
}

func (self *loggerInstrument) allocate(ctx string) {
	logrus.WithField("context", ctx).Warn("allocate")
}

func (self *loggerInstrument) configure(data map[interface{}]interface{}) error {
	return nil
}

func (self *loggerInstrument) symbol(mt messageType) rune {
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
