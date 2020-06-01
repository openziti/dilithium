package westworld2

import (
	"github.com/emirpasic/gods/trees/btree"
	"github.com/emirpasic/gods/utils"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"net"
)

type txPortal struct {
	tree     *btree.Tree
	capacity int
	txQueue  chan *wireMessage
	txErrors chan error
	rxAcks   chan int32
	txAcks   chan int32
	conn     *net.UDPConn
	peer     *net.UDPAddr
	pool     *pool
}

func newTxPortal(conn *net.UDPConn, peer *net.UDPAddr) *txPortal {
	txp := &txPortal{
		tree:     btree.NewWith(treeSize, utils.Int32Comparator),
		capacity: startingWindowCapacity,
		txQueue:  make(chan *wireMessage, txQueueSize),
		txErrors: make(chan error, txErrorsSize),
		rxAcks:   make(chan int32, ackQueueSize),
		txAcks:   make(chan int32, ackQueueSize),
		conn:     conn,
		peer:     peer,
		pool:     newPool("txPortal"),
	}
	go txp.run()
	return txp
}

func (self *txPortal) run() {
	logrus.Info("started")
	defer logrus.Warn("exited")

	for {
		if err := self.rxtxAcks(); err != nil {
			logrus.Errorf("rxtxAcks (%v)", err)
			return
		}
		if err:= self.txData(); err != nil {
			logrus.Errorf("txData (%v)", err)
		}
	}
}

func (self *txPortal) rxtxAcks() error {
	select {
	case txAck, ok := <-self.txAcks:
		if !ok {
			return errors.New("txAcks closed")
		}

		wm := newAck(txAck, self.pool)
		if err := writeWireMessage(wm, self.conn, self.peer); err != nil {
			logrus.Errorf("{@%d} -> (%v)", txAck, err)
		}
		wm.buffer.unref()

	case rxAck, ok := <-self.rxAcks:
		if !ok {
			return errors.New("rxAcks closed")
		}

		if wm, found := self.tree.Get(rxAck); found {
			self.tree.Remove(rxAck)
			wm.(*wireMessage).buffer.unref()
			self.capacity++

		} else {
			logrus.Warnf("~ [@%d] <-", rxAck) // already acked
		}

	default:
		// no acks to process
	}
	return nil
}

func (self *txPortal) txData() error {
	if self.capacity > 0 {
		select {
		case wm, ok := <-self.txQueue:
			if !ok {
				return errors.New("txQueue closed")
			}

			self.tree.Put(wm.seq, wm)
			self.capacity--

			select {
			case ack, ok := <-self.txAcks:
				if !ok {
					return errors.New("txAcks closed")
				}
				wm.rewriteAck(ack)
			default:
				// no ack for slipstream
			}

			if err := writeWireMessage(wm, self.conn, self.peer); err != nil {
				logrus.Errorf("-> {#%d,@%d}[%d] -> (%v)", wm.seq, wm.ack, len(wm.data), err)
				self.txErrors <- errors.Wrap(err, "write")
			}

		default:
			// no wireMessage for tx
		}
	}
	return nil
}
