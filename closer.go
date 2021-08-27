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

const notClosed = int32(-99)
