package westworld2

import (
	"github.com/sirupsen/logrus"
	"net"
)

type instrument interface {
	accepted(peer *net.UDPAddr)
	wireMessageTx(wm *wireMessage)
	wireMessageRx(wm *wireMessage)
	unknownPeer(peer *net.UDPAddr)
	readError(peer *net.UDPAddr, err error)
	connectError(peer *net.UDPAddr, err error)
}

type loggerInstrument struct{}

func (self *loggerInstrument) accepted(peer *net.UDPAddr) {
	logrus.Infof("accepted, peer [%s]", peer)
}

func (self *loggerInstrument) wireMessageRx(wm *wireMessage) {
	logrus.Infof("<- [{%c},#%d,@%d] <-", self.symbol(wm.mt), wm.seq, wm.ack)
}

func (self *loggerInstrument) wireMessageTx(wm *wireMessage) {
	logrus.Infof("-> [{%c},#%d,@%d] ->", self.symbol(wm.mt), wm.seq, wm.ack)
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