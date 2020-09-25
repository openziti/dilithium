package westworld2

import (
	"github.com/sirupsen/logrus"
	"net"
)

type loggerInstrument struct{}

func newLoggerInstrument() Instrument {
	return &loggerInstrument{}
}

func (self *loggerInstrument) newInstance(peer *net.UDPAddr) InstrumentInstance {
	return &loggerInstrumentInstance{peer: peer}
}

type loggerInstrumentInstance struct{
	peer *net.UDPAddr
}

/*
 * connection
 */
func (self *loggerInstrumentInstance) connected(peer *net.UDPAddr) {
	logrus.Infof("connected, peer [%s]", peer)
}

func (self *loggerInstrumentInstance) closed(peer *net.UDPAddr) {
	logrus.Infof("closed, peer [%s]", peer)
}

func (self *loggerInstrumentInstance) connectError(peer *net.UDPAddr, err error) {
	logrus.Errorf("connect failed, peer [%s] (%v)", peer, err)
}
/* */

/*
 * wire
 */
func (self *loggerInstrumentInstance) wireMessageRx(peer *net.UDPAddr, wm *wireMessage) {
	logrus.Infof("<- [%c/#%d/@%d/:%d] <- [%s]", self.symbol(wm.mt), wm.seq, wm.ack, len(wm.data), peer)
}

func (self *loggerInstrumentInstance) wireMessageTx(peer *net.UDPAddr, wm *wireMessage) {
	logrus.Infof("-> [%c/#%d/@%d/:%d] -> [%s]", self.symbol(wm.mt), wm.seq, wm.ack, len(wm.data), peer)
}

func (self *loggerInstrumentInstance) wireMessageRetx(peer *net.UDPAddr, wm *wireMessage) {
	logrus.Warnf("!> [%c/#%d/@%d/:%d] -> [%s]", self.symbol(wm.mt), wm.seq, wm.ack, len(wm.data), peer)
}

func (self *loggerInstrumentInstance) unknownPeer(peer *net.UDPAddr) {
	logrus.Errorf("unknown peer [%s]", peer)
}

func (self *loggerInstrumentInstance) readError(peer *net.UDPAddr, err error) {
	logrus.Errorf("read error, peer [%s] (%v)", peer, err)
}

func (self *loggerInstrumentInstance) unexpectedMessageType(peer *net.UDPAddr, mt messageType) {
	logrus.Errorf("unexpected message type [%s], peer [%s]", mt.string(), peer)
}
/* */

/*
 * txPortal
 */
func (self *loggerInstrumentInstance) txPortalCapacityChanged(_ *net.UDPAddr, _ int) {
}

func (self *loggerInstrumentInstance) txPortalSzChanged(_ *net.UDPAddr, _ int) {
}

func (self *loggerInstrumentInstance) txPortalRxPortalSzChanged(_ *net.UDPAddr, _ int) {
}

func (self *loggerInstrumentInstance) newRetxMs(peer *net.UDPAddr, retxMs int) {
	logrus.Infof("!+[%d ms]  <- [%s]", retxMs, peer)
}

func (self *loggerInstrumentInstance) duplicateAck(peer *net.UDPAddr, ack int32) {
	logrus.Warnf("~ <- [@%d] <- [%s]", ack, peer)
}
/* */

/*
 * rxPortal
 */
func (self *loggerInstrumentInstance) rxPortalSzChanged(_ *net.UDPAddr, _ int) {
}

func (self *loggerInstrumentInstance) duplicateRx(peer *net.UDPAddr, wm *wireMessage) {
	logrus.Warnf("~ <- [#%d] <- [%s]", wm.seq, peer)
}
/* */

/*
 * allocation
 */
func (self *loggerInstrumentInstance) allocate(ctx string) {
	logrus.WithField("context", ctx).Warn("allocate")
}
/* */

func (self *loggerInstrumentInstance) symbol(mt messageType) rune {
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
