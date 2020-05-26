package westworld

import (
	"github.com/emirpasic/gods/trees/btree"
	"github.com/emirpasic/gods/utils"
	"github.com/michaelquigley/dilithium/protocol/westworld/pb"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"net"
	"sync"
)

type txWindow struct {
	lock              *sync.Mutex
	tree              *btree.Tree
	capacity          int
	capacityAvailable *sync.Cond
	txQueue           chan *pb.WireMessage
	txErrors		  chan error
	ackQueue          chan int32
	conn              *net.UDPConn
	peer              *net.UDPAddr
}

func newTxWindow(ackQueue chan int32, conn *net.UDPConn, peer *net.UDPAddr) *txWindow {
	txw := &txWindow{
		lock:       new(sync.Mutex),
		tree:       btree.NewWith(startingTreeSize, utils.Int32Comparator),
		capacity:   startingWindowCapacity,
		txQueue:	make(chan *pb.WireMessage, startingTreeSize),
		txErrors:	make(chan error, startingTreeSize),
		ackQueue:   ackQueue,
		conn:       conn,
		peer:       peer,
	}
	txw.capacityAvailable = sync.NewCond(txw.lock)
	go txw.txer()
	return txw
}

func (self *txWindow) tx(wm *pb.WireMessage) {
	self.lock.Lock()
	defer self.lock.Unlock()

	for self.capacity < 1 {
		self.capacityAvailable.Wait()
	}

	self.tree.Put(wm.Sequence, wm)
	self.capacity--

	self.txQueue <- wm

	logrus.Infof("queued")
}

func (self *txWindow) ack(sequence int32) {
	self.lock.Lock()
	defer self.lock.Unlock()

	if _, found := self.tree.Get(sequence); found {
		self.tree.Remove(sequence)
		self.capacity++
		self.capacityAvailable.Signal()

	} else {
		logrus.Warnf("~ <- [@#%d] <-", sequence) // already acked
	}
}

func (self *txWindow) txer() {
	logrus.Infof("started")
	defer logrus.Warnf("exited")

	for {
		select {
		case wm, ok := <-self.txQueue:
			if !ok {
				return
			}
			ackSequence := int32(-1)
			select {
			case sequence := <-self.ackQueue:
				ackSequence = sequence
			default:
			}
			wm.Ack = ackSequence

			if err := pb.WriteWireMessage(wm, self.conn, self.peer); err == nil {
				logrus.Infof("-> {#%d,@%d}[%d] ->", wm.Sequence, wm.Ack, len(wm.Data))

			} else {
				logrus.Errorf("-> {#%d,@%d)[%d] -> (%v)", wm.Sequence, wm.Ack, len(wm.Data), err)
				self.txErrors <- errors.Wrap(err, "write")
			}

		case sequence, ok := <-self.ackQueue:
			if !ok {
				return
			}
			wm := pb.NewAck(sequence)
			if err := pb.WriteWireMessage(wm, self.conn, self.peer); err == nil {
				logrus.Infof("{@%d} ->", sequence)

			} else {
				logrus.Errorf("{@%d} -> (%v)", sequence, err)
				self.txErrors <- errors.Wrap(err, "write ack")
			}
		}
	}
}