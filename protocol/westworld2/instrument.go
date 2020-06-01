package westworld2

import "github.com/sirupsen/logrus"

type instrument interface {
	wireMessageTx(wm *wireMessage)
	wireMessageRx(wm *wireMessage)
}

type loggerInstrument struct{}

func (self *loggerInstrument) wireMessageRx(wm *wireMessage) {
	logrus.Infof("<- {%s}[#%d] <-", mtString(wm.mt), wm.seq)
}

func (self *loggerInstrument) wireMessageTx(wm *wireMessage) {
	logrus.Infof("-> {%s}[#%d] ->", mtString(wm.mt), wm.seq)
}
