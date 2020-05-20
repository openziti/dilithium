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
	conn              *net.UDPConn
	peer              *net.UDPAddr
}

type txWindowMarshalled struct {
	wm   *pb.WireMessage
	data []byte
}

func newTxWindow(conn *net.UDPConn, peer *net.UDPAddr) *txWindow {
	txw := &txWindow{
		lock:     new(sync.Mutex),
		tree:     btree.NewWith(startingTreeSize, utils.Int32Comparator),
		capacity: startingWindowCapacity,
		conn:     conn,
		peer:     peer,
	}
	txw.capacityAvailable = sync.NewCond(txw.lock)
	return txw
}

func (self *txWindow) tx(txm *txWindowMarshalled) {
	self.lock.Lock()
	defer self.lock.Unlock()

	for self.capacity < 1 {
		self.capacityAvailable.Wait()
	}

	// opportunity to ack-stamp our outgoing
	self.tree.Put(txm.wm.Sequence, txm)
	self.capacity--
}

func (self *txWindow) ack(sequence int32) {
	self.lock.Lock()
	defer self.lock.Unlock()

	if txm, found := self.tree.Get(sequence); found {
		self.tree.Remove(sequence)
		self.capacity++
		self.capacityAvailable.Signal()

	} else {
		logrus.Warnf("~ <- [@#%d] <-", sequence) // already acked
	}
}