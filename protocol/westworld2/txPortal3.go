package westworld2

import (
	"github.com/emirpasic/gods/trees/btree"
	"github.com/emirpasic/gods/utils"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"net"
	"sort"
	"sync"
	"time"
)

type txPortal3 struct {
	lock      *sync.Mutex
	tree      *btree.Tree
	capacity  int
	ready     *sync.Cond
	txMonitor *txMonitor
	conn      *net.UDPConn
	peer      *net.UDPAddr
	pool      *pool
	ins       Instrument
}

type txMonitor struct {
	waiting   []*txMonitored
	head      *txMonitored
	cancelled bool
	ready     *sync.Cond
}

type txMonitored struct {
	deadline time.Time
	wm       *wireMessage
}

func newTxPortal3(conn *net.UDPConn, peer *net.UDPAddr, ins Instrument) *txPortal3 {
	txp := &txPortal3{
		lock:      new(sync.Mutex),
		tree:      btree.NewWith(treeSize, utils.Int32Comparator),
		capacity:  startingWindowCapacity,
		txMonitor: &txMonitor{},
		conn:      conn,
		peer:      peer,
		pool:      newPool("txPortal", ins),
		ins:       ins,
	}
	txp.ready = sync.NewCond(txp.lock)
	txp.txMonitor.ready = sync.NewCond(txp.lock)
	go txp.monitor()
	return txp
}

func (self *txPortal3) tx(wm *wireMessage) error {
	self.lock.Lock()
	defer self.lock.Unlock()

	for self.capacity < 1 {
		self.ready.Wait()
	}

	self.tree.Put(wm.seq, wm)
	self.capacity--
	self.addMonitor(wm)

	if err := writeWireMessage(wm, self.conn, self.peer, self.ins); err != nil {
		return errors.Wrap(err, "tx")
	}

	return nil
}

func (self *txPortal3) ack(sequence int32) {
	self.lock.Lock()
	defer self.lock.Unlock()

	if wm, found := self.tree.Get(sequence); found {
		self.cancelMonitor(wm.(*wireMessage))
		self.tree.Remove(sequence)
		wm.(*wireMessage).buffer.unref()
		self.capacity++
		self.ready.Signal()
	} else {
		if self.ins != nil {
			self.ins.duplicateAck(self.peer, sequence)
		}
	}
}

func (self *txPortal3) monitor() {
	logrus.Info("started")
	defer logrus.Warn("exited")

	for {
		var timeout time.Duration

		self.lock.Lock()
		for len(self.txMonitor.waiting) < 1 {
			self.txMonitor.ready.Wait()
		}
		self.txMonitor.head = self.txMonitor.waiting[0]
		timeout = time.Until(self.txMonitor.head.deadline)
		self.txMonitor.cancelled = false
		self.lock.Unlock()

		time.Sleep(timeout)

		self.lock.Lock()
		if !self.txMonitor.cancelled {
			if self.ins != nil {
				self.ins.wireMessageRetx(self.peer, self.txMonitor.head.wm)
			}
			if err := writeWireMessage(self.txMonitor.head.wm, self.conn, self.peer, self.ins); err != nil {
				logrus.Errorf("retx (%v)", err)
			}
			self.txMonitor.head.deadline = time.Now().Add(retxTimeoutMs * time.Millisecond)
			sort.Slice(self.txMonitor.waiting, func(i, j int) bool {
				return self.txMonitor.waiting[i].deadline.Before(self.txMonitor.waiting[j].deadline)
			})
		}
		self.lock.Unlock()
	}
}

func (self *txPortal3) addMonitor(wm *wireMessage) {
	wm.buffer.ref()
	self.txMonitor.waiting = append(self.txMonitor.waiting, &txMonitored{time.Now().Add(retxTimeoutMs * time.Millisecond), wm})
	self.txMonitor.ready.Signal()
}

func (self *txPortal3) cancelMonitor(wm *wireMessage) {
	i := -1
	for j, m := range self.txMonitor.waiting {
		if m.wm == wm {
			i = j
			break
		}
	}
	if i > -1 {
		self.txMonitor.waiting = append(self.txMonitor.waiting[:i], self.txMonitor.waiting[i+1:]...)
	}
	if self.txMonitor.head != nil && self.txMonitor.head.wm == wm {
		self.txMonitor.cancelled = true
	}
}
