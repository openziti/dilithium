package westworld

import (
	"github.com/emirpasic/gods/trees/btree"
	"github.com/emirpasic/gods/utils"
	"github.com/michaelquigley/dilithium/protocol/westworld/pb"
	"github.com/sirupsen/logrus"
	"net"
	"sync"
)

type txWindow struct {
	lock              *sync.Mutex
	tree              *btree.Tree
	capacity          int
	capacityAvailable *sync.Cond
	ackQueue          chan int32
	conn              *net.UDPConn
	peer              *net.UDPAddr
}

func newTxWindow(conn *net.UDPConn, peer *net.UDPAddr) *txWindow {
	txw := &txWindow{
		lock:     new(sync.Mutex),
		tree:     btree.NewWith(startingTreeSize, utils.Int32Comparator),
		capacity: startingWindowCapacity,
		ackQueue: make(chan int32, ackQueueLength),
		conn:     conn,
		peer:     peer,
	}
	txw.capacityAvailable = sync.NewCond(txw.lock)
	return txw
}

func (self *txWindow) tx(wm *pb.WireMessage) {
	self.lock.Lock()
	defer self.lock.Unlock()

	for self.capacity < 1 {
		self.capacityAvailable.Wait()
	}

	select {
	case sequence := <-self.ackQueue:
		// stamp the pending ack
		wm.Ack = sequence
	default:
	}

	self.tree.Put(wm.Sequence, wm)
	self.capacity--
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
