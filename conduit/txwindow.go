package conduit

import (
	"github.com/emirpasic/gods/trees/btree"
	"github.com/emirpasic/gods/utils"
	"github.com/sirupsen/logrus"
	"net"
	"sync"
)

const startingWindowCapacity = 8

type txWindow struct {
	tree          *btree.Tree
	lock          *sync.Mutex
	capacity      int
	capacityReady *sync.Cond
	conn          *net.UDPConn
	peer          *net.UDPAddr
}

func newTxWindow(conn *net.UDPConn, peer *net.UDPAddr) *txWindow {
	txw := &txWindow{
		tree:     btree.NewWith(10240, utils.Int32Comparator),
		lock:     new(sync.Mutex),
		capacity: startingWindowCapacity,
		conn:     conn,
		peer:     peer,
	}
	txw.capacityReady = sync.NewCond(txw.lock)
	return txw
}

func (self *txWindow) tx(m *message) {
	self.lock.Lock()
	defer self.lock.Unlock()

	for self.capacity < 1 {
		self.capacityReady.Wait()
	}

	self.tree.Put(m.sequence, m)
	self.capacity--
}

func (self *txWindow) ack(sequence int32) {
	self.lock.Lock()
	defer self.lock.Unlock()

	if _, found := self.tree.Get(sequence); found {
		self.tree.Remove(sequence)
		self.capacity++
		self.capacityReady.Broadcast()
	} else {
		logrus.Warnf("already acked [#%d]", sequence)
	}
}
