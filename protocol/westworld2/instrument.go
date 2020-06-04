package westworld2

import (
	"github.com/sirupsen/logrus"
	"net"
)

type instrument interface {
	connected(peer *net.UDPAddr)
	wireMessageTx(peer *net.UDPAddr, wm *wireMessage)
	wireMessageRx(peer *net.UDPAddr, wm *wireMessage)
	unknownPeer(peer *net.UDPAddr)
	readError(peer *net.UDPAddr, err error)
	connectError(peer *net.UDPAddr, err error)
	unexpectedMessageType(peer *net.UDPAddr, mt messageType)
}

type loggerInstrument struct{}

func (self *loggerInstrument) connected(peer *net.UDPAddr) {
	logrus.Infof("connected, peer [%s]", peer)
}

func (self *loggerInstrument) wireMessageRx(peer *net.UDPAddr, wm *wireMessage) {
	logrus.Infof("<- [{%c},#%d,@%d] <- [%s]", self.symbol(wm.mt), wm.seq, wm.ack, peer)
}

func (self *loggerInstrument) wireMessageTx(peer *net.UDPAddr, wm *wireMessage) {
	logrus.Infof("-> [{%c},#%d,@%d] -> [%s]", self.symbol(wm.mt), wm.seq, wm.ack, peer)
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
	logrus.Errorf("unexpected message type [%s], peer [%s]", mtString(mt), peer)
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