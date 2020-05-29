package westworld2

import (
	"github.com/emirpasic/gods/trees/btree"
	"github.com/emirpasic/gods/utils"
	"github.com/sirupsen/logrus"
	"net"
)

type txWindow struct {
	tree     *btree.Tree
	capacity int
	txQueue  chan *wireMessage
	txErrors chan error
	ackQueue chan int32
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
		ackQueue: make(chan int32, ackQueueSize),
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
				case ack, ok := <-self.ackQueue:
					if !ok {
						return
					}
					wm.rewriteAck(ack)
				default:
				}

			default:
			}
		}
	}
}