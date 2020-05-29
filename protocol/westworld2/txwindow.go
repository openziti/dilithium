package westworld2

import (
	"github.com/emirpasic/gods/trees/btree"
	"github.com/emirpasic/gods/utils"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"net"
)

type txWindow struct {
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

func newTxWindow(ackQueue chan int32, conn *net.UDPConn, peer *net.UDPAddr) *txWindow {
	return &txWindow{
		tree:     btree.NewWith(treeSize, utils.Int32Comparator),
		capacity: startingWindowCapacity,
		txQueue:  make(chan *wireMessage, txQueueSize),
		txErrors: make(chan error, txErrorsSize),
		rxAcks:   make(chan int32, ackQueueSize),
		txAcks:   make(chan int32, ackQueueSize),
		conn:     conn,
		peer:     peer,
		pool:     newPool("txWindow"),
	}
}

func (self *txWindow) run() {
	logrus.Info("started")
	defer logrus.Warn("exited")

	for {
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

			case txAck, ok := <-self.txAcks:
				if !ok {
					return
				}

				wm := newAck(txAck, self.pool.get())
				if err := writeWireMessage(wm, self.conn, self.peer); err != nil {
					logrus.Errorf("{@%d} -> (%v)", txAck, err)
				}
				wm.buffer.unref()

			default:
				// no wireMessage for tx
			}
		}
		
		select {
		case rxAck, ok := <- self.rxAcks:
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
}
