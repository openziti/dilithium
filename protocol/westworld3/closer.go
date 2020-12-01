package westworld3

import (
	"github.com/openziti/dilithium/util"
	"github.com/sirupsen/logrus"
	"time"
)

const notClosed = int32(-33)

type closer struct {
	seq          *util.Sequence
	closee       bool
	rxCloseSeq   int32
	rxCloseSeqIn chan int32
	txCloseSeq   int32
	txCloseSeqIn chan int32
	txPortal     *txPortal
	rxPortal     *rxPortal
	lastEvent    time.Time
	profile      *Profile
	closeHook    func()
}

func newCloser(seq *util.Sequence, profile *Profile, closeHook func()) *closer {
	return &closer{
		seq:          seq,
		rxCloseSeq:   notClosed,
		rxCloseSeqIn: make(chan int32, 1),
		txCloseSeq:   notClosed,
		txCloseSeqIn: make(chan int32, 1),
		profile:      profile,
		closeHook:    closeHook,
	}
}

func (self *closer) emergencyStop() {
	logrus.Infof("broken glass")

	self.txPortal.close()
	self.rxPortal.close()

	if self.closeHook != nil {
		self.closeHook()
	}
}

func (self *closer) run() {
closeWait:
	for {
		select {
		case rxCloseSeq, ok := <-self.rxCloseSeqIn:
			if !ok {
				logrus.Info("unexpected closed rx close seq")
				break closeWait
			}
			self.rxCloseSeq = rxCloseSeq
			self.lastEvent = time.Now()
			logrus.Infof("got rx close seq: %d", rxCloseSeq)
			if self.txCloseSeq == notClosed {
				self.closee = true
				if err := self.txPortal.sendClose(self.seq); err != nil {
					logrus.Errorf("error sending close (%v)", err)
				}
			}
			if self.readyToClose() {
				break closeWait
			}

		case txCloseSeq, ok := <-self.txCloseSeqIn:
			if !ok {
				logrus.Infof("unexpected closed tx close seq")
				break closeWait
			}
			self.txCloseSeq = txCloseSeq
			self.lastEvent = time.Now()
			logrus.Infof("got tx close seq: %d", txCloseSeq)
			if self.readyToClose() {
				break closeWait
			}

		case <-time.After(time.Duration(self.profile.CloseCheckMs) * time.Millisecond):
			if self.readyToClose() {
				break closeWait
			}
		}
	}
	logrus.Info("ready to close")

	self.txPortal.close()
	self.rxPortal.close()

	if self.closeHook != nil {
		self.closeHook()
	}

	logrus.Info("close complete")
}

func (self *closer) readyToClose() bool {
	return self.txCloseSeq != notClosed && self.rxCloseSeq != notClosed && time.Since(self.lastEvent).Milliseconds() > int64(self.profile.CloseWaitMs)
}
