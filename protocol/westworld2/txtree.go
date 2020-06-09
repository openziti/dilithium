package westworld2

import (
	"github.com/emirpasic/gods/trees/btree"
	"github.com/emirpasic/gods/utils"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"net"
)

const txTreeSz = 1024

type txTree struct {
	acks        chan int32
	txs         chan *wireMessage
	errors      chan error
	retxMonitor chan *wireMessage
	retxCancel  chan int32
	tree        *btree.Tree
	capacity    int
	conn        *net.UDPConn
	peer        *net.UDPAddr
	ins         Instrument
}

func newTxTree(retxMonitor chan *wireMessage, retxCancel chan int32, conn *net.UDPConn, peer *net.UDPAddr, ins Instrument) *txTree {
	tt := &txTree{
		acks:        make(chan int32, txTreeSz),
		txs:         make(chan *wireMessage, txTreeSz),
		errors:      make(chan error, txTreeSz),
		retxMonitor: retxMonitor,
		retxCancel:  retxCancel,
		tree:        btree.NewWith(txTreeSz, utils.Int32Comparator),
		capacity:    startingWindowCapacity,
		conn:        conn,
		peer:        peer,
		ins:         ins,
	}
	go tt.run()
	return tt
}

func (self *txTree) run() {
	for {
	acks:
		for {
			select {
			case acked, ok := <-self.acks:
				if !ok {
					return
				}

				self.retxCancel <- acked
				if _, found := self.tree.Get(acked); found {
					self.tree.Remove(acked)
					//wm.(*wireMessage).buffer.unref()
					self.capacity++

				} else {
					logrus.Warnf("~ [@%d] <- (capacity: %d)", acked, self.capacity)
				}

			default:
				break acks
			}
		}

	txs:
		for self.capacity > 0 {
			select {
			case wm, ok := <-self.txs:
				if !ok {
					return
				}

				self.tree.Put(wm.seq, wm)
				self.capacity--
				self.retxMonitor <- wm

				if err := writeWireMessage(wm, self.conn, self.peer, self.ins); err != nil {
					logrus.Errorf("-> [#%d] -> (%v)", wm.seq, err)
					self.errors <- errors.Wrap(err, "write")
				}

			default:
				break txs
			}
		}
	}
}
