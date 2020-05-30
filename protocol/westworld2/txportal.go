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

func newTxPortal(ackQueue chan int32, conn *net.UDPConn, peer *net.UDPAddr) *txPortal {
	return &txPortal{
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
}

func (self *txPortal) run() {
	logrus.Info("started")
	defer logrus.Warn("exited")

	for {
		self.rxtxAcks()
		self.txData()
	}
}

func (self *txPortal) rxtxAcks() {
	select {
	case txAck, ok := <-self.txAcks:
		if !ok {
			return
		}

		wm := newAck(txAck, self.pool.get())
		if err := writeWireMessage(wm, self.conn, self.peer); err != nil {
			logrus.Errorf("{@%d} -> (%v)", txAck, err)
		}
		wm.buffer.unref()

	case rxAck, ok := <-self.rxAcks:
		if !ok {
			return
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
}

func (self *txPortal) txData() {
	if self.capacity > 0 {
		select {
		case wm, ok := <-self.txQueue:
			if !ok {
				return
			}

			self.tree.Put(wm.seq, wm)
			self.capacity--

			select {
			case ack, ok := <-self.txAcks:
				if !ok {
					return
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
}
