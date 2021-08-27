package dilithium

import (
	"github.com/openziti/dilithium/util"
	"github.com/sirupsen/logrus"
	"time"
)

// closer manages the state machine for the shutdown of a TxPortal and RxPortal pair (one side of a communication).
//
type closer struct {
	seq          *util.Sequence
	rxCloseSeq   int32
	rxCloseSeqIn chan int32
	txCloseSeq   int32
	txCloseSeqIn chan int32
	txp          *TxPortal
	rxp          *RxPortal
	lastEvent    time.Time
	closeHook    func()
}

func newCloser(seq *util.Sequence, closeHook func()) *closer {
	return &closer{
		seq:          seq,
		rxCloseSeq:   notClosed,
		rxCloseSeqIn: make(chan int32, 1),
		txCloseSeq:   notClosed,
		txCloseSeqIn: make(chan int32, 1),
		closeHook:    closeHook,
	}
}

func (c *closer) emergencyStop() {
	logrus.Info("broken glass")
	c.txp.close()
	c.rxp.Close()
	if c.closeHook != nil {
		c.closeHook()
	}
}

func (c *closer) timeout() {
	logrus.Info("timeout")

	c.txp.close()
	c.rxp.Close()

	if c.closeHook != nil {
		c.closeHook()
	}
}

func (c *closer) run() {
	logrus.Info("started")
	defer logrus.Info("exited")

closeWait:
	for {
		select {
		case rxCloseSeq, ok := <-c.rxCloseSeqIn:
			if !ok {
				logrus.Info("!rx close seq")
				break closeWait
			}
			c.rxCloseSeq = rxCloseSeq
			c.lastEvent = time.Now()
			logrus.Infof("got rx close seq [%d]", rxCloseSeq)
			if c.txCloseSeq == notClosed {
				if err := c.txp.sendClose(c.seq); err != nil {
					logrus.Errorf("error sending close (%v)", err)
				}
			}
			if c.readyToClose() {
				break closeWait
			}

		case txCloseSeq, ok := <-c.txCloseSeqIn:
			if !ok {
				logrus.Infof("!tx close seq")
				break closeWait
			}
			c.txCloseSeq = txCloseSeq
			c.lastEvent = time.Now()
			logrus.Infof("got tx close seq [%d]", txCloseSeq)
			if c.readyToClose() {
				break closeWait
			}

		case <-time.After(time.Duration(c.txp.alg.Profile().CloseCheckMs) * time.Millisecond):
			if c.readyToClose() {
				break closeWait
			}
		}
	}
	logrus.Info("ready to close")

	c.txp.close()
	c.rxp.Close()

	if c.closeHook != nil {
		c.closeHook()
	}

	logrus.Info("close complete")
}

func (c *closer) readyToClose() bool {
	return (c.txCloseSeq != notClosed && c.rxCloseSeq != notClosed) || time.Since(c.lastEvent).Milliseconds() > int64(c.txp.alg.Profile().ConnectionTimeout)
}

const notClosed = int32(-99)
