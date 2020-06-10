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

type txPortal struct {
	lock     *sync.Mutex
	tree     *btree.Tree
	capacity int
	ready    *sync.Cond
	monitor  *retxMonitor
	conn     *net.UDPConn
	peer     *net.UDPAddr
	pool     *pool
	ins      Instrument
}

type retxMonitor struct {
	waiting   []*retxSubject
	head      *retxSubject
	cancelled bool
	ready     *sync.Cond
}

type retxSubject struct {
	deadline time.Time
	wm       *wireMessage
}

func newTxPortal(conn *net.UDPConn, peer *net.UDPAddr, ins Instrument) *txPortal {
	txp := &txPortal{
		lock:     new(sync.Mutex),
		tree:     btree.NewWith(treeSz, utils.Int32Comparator),
		capacity: startingWindowCapacity,
		monitor:  &retxMonitor{},
		conn:     conn,
		peer:     peer,
		pool:     newPool("txPortal", ins),
		ins:      ins,
	}
	txp.ready = sync.NewCond(txp.lock)
	txp.monitor.ready = sync.NewCond(txp.lock)
	go txp.runMonitor()
	return txp
}

func (self *txPortal) tx(wm *wireMessage) error {
	sz := len(wm.data)

	self.lock.Lock()
	defer self.lock.Unlock()

	for self.capacity < sz {
		self.ready.Wait()
	}

	self.tree.Put(wm.seq, wm)
	self.capacity -= sz
	self.addMonitor(wm)

	if err := writeWireMessage(wm, self.conn, self.peer, self.ins); err != nil {
		return errors.Wrap(err, "tx")
	}

	return nil
}

func (self *txPortal) ack(sequence int32) {
	self.lock.Lock()
	defer self.lock.Unlock()

	if v, found := self.tree.Get(sequence); found {
		wm := v.(*wireMessage)
		self.cancelMonitor(wm)
		self.tree.Remove(sequence)
		self.capacity += len(wm.data)
		wm.buffer.unref()
		self.ready.Signal()

	} else {
		if self.ins != nil {
			self.ins.duplicateAck(self.peer, sequence)
		}
	}
}

func (self *txPortal) runMonitor() {
	logrus.Info("started")
	defer logrus.Warn("exited")

	for {
		var timeout time.Duration

		self.lock.Lock()
		for len(self.monitor.waiting) < 1 {
			self.monitor.ready.Wait()
		}
		self.monitor.head = self.monitor.waiting[0]
		timeout = time.Until(self.monitor.head.deadline)
		self.monitor.cancelled = false
		self.lock.Unlock()

		time.Sleep(timeout)

		self.lock.Lock()
		if !self.monitor.cancelled {
			if self.ins != nil {
				self.ins.wireMessageRetx(self.peer, self.monitor.head.wm)
			}
			if err := writeWireMessage(self.monitor.head.wm, self.conn, self.peer, self.ins); err != nil {
				logrus.Errorf("retx (%v)", err)
			}
			self.monitor.head.deadline = time.Now().Add(retxTimeoutMs * time.Millisecond)
			sort.Slice(self.monitor.waiting, func(i, j int) bool {
				return self.monitor.waiting[i].deadline.Before(self.monitor.waiting[j].deadline)
			})
		}
		self.lock.Unlock()
	}
}

func (self *txPortal) addMonitor(wm *wireMessage) {
	wm.buffer.ref()
	self.monitor.waiting = append(self.monitor.waiting, &retxSubject{time.Now().Add(retxTimeoutMs * time.Millisecond), wm})
	self.monitor.ready.Signal()
}

func (self *txPortal) cancelMonitor(wm *wireMessage) {
	i := -1
	for j, m := range self.monitor.waiting {
		if m.wm == wm {
			i = j
			break
		}
	}
	if i > -1 {
		self.monitor.waiting[i].wm.buffer.unref()
		self.monitor.waiting = append(self.monitor.waiting[:i], self.monitor.waiting[i+1:]...)
	}
	if self.monitor.head != nil && self.monitor.head.wm == wm {
		self.monitor.cancelled = true
	}
}
