package westworld2

import "github.com/sirupsen/logrus"

type instrument interface {
	wireMessageTx(wm *wireMessage)
	wireMessageRx(wm *wireMessage)
}

type loggerInstrument struct{}

func (self *loggerInstrument) wireMessageRx(wm *wireMessage) {
	logrus.Infof("<- [{%c},#%d,@%d] <-", self.symbol(wm.mt), wm.seq, wm.ack)
}

func (self *loggerInstrument) wireMessageTx(wm *wireMessage) {
	logrus.Infof("-> [{%c},#%d,@%d] ->", self.symbol(wm.mt), wm.seq, wm.ack)
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