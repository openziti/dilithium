package westworld

import (
	"github.com/emirpasic/gods/trees/btree"
	"github.com/emirpasic/gods/utils"
	"github.com/michaelquigley/dilithium/protocol/westworld/wb"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"net"
	"sync"
	"time"
)

type txWindow struct {
	lock              *sync.Mutex
	tree              *btree.Tree
	capacity          int
	capacityAvailable *sync.Cond
	txMonitor         *txMonitor
	txQueue           chan *wb.WireMessage
	txErrors          chan error
	ackQueue          chan int32
	conn              *net.UDPConn
	peer              *net.UDPAddr
	pool              *wb.BufferPool
}

type txMonitor struct {
	waiting    []*txMonitored
	monitoring *txMonitored
	cancelled  bool
	ready      *sync.Cond
}

type txMonitored struct {
	timeout time.Time
	retries int
	wm      *wb.WireMessage
}

func newTxWindow(ackQueue chan int32, conn *net.UDPConn, peer *net.UDPAddr) *txWindow {
	txw := &txWindow{
		lock:      new(sync.Mutex),
		tree:      btree.NewWith(startingTreeSize, utils.Int32Comparator),
		capacity:  startingWindowCapacity,
		txMonitor: &txMonitor{},
		txQueue:   make(chan *wb.WireMessage, startingTreeSize),
		txErrors:  make(chan error, startingTreeSize),
		ackQueue:  ackQueue,
		conn:      conn,
		peer:      peer,
		pool:      wb.NewBufferPool("txWindow"),
	}
	txw.capacityAvailable = sync.NewCond(txw.lock)
	txw.txMonitor.ready = sync.NewCond(txw.lock)
	go txw.txer()
	go txw.monitor()
	return txw
}

func (self *txWindow) tx(wm *wb.WireMessage) {
	self.lock.Lock()
	defer self.lock.Unlock()

	for self.capacity < 1 {
		self.capacityAvailable.Wait()
	}

	self.tree.Put(wm.Sequence, wm)
	self.capacity--
	self.addMonitor(wm)

	self.txQueue <- wm
}

func (self *txWindow) ack(sequence int32) {
	self.lock.Lock()
	defer self.lock.Unlock()

	if wm, found := self.tree.Get(sequence); found {
		self.cancelMonitor(wm.(*wb.WireMessage))
		self.tree.Remove(sequence)
		wm.(*wb.WireMessage).Unref()
		self.capacity++
		self.capacityAvailable.Signal()

	} else {
		logrus.Warnf("~ <- [@%d] <-", sequence) // already acked
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

			select {
			case ackSequence := <-self.ackQueue:
				if err := wm.RewriteAck(ackSequence); err == nil {
					//logrus.Warnf("sliplined ack {@%d}", ackSequence)
				} else {
					logrus.Errorf("error rewriting ack (%v)", err)
				}

			default:
			}

			if err := wm.WriteMessage(self.conn, self.peer); err == nil {
				//logrus.Infof("-> {#%d,@%d}[%d] ->", wm.Sequence, wm.Ack, len(wm.Data))

			} else {
				logrus.Errorf("-> {#%d,@%d)[%d] -> (%v)", wm.Sequence, wm.Ack, len(wm.Data), err)
				self.txErrors <- errors.Wrap(err, "write")
			}

		case sequence, ok := <-self.ackQueue:
			if !ok {
				return
			}
			if wm, err := wb.NewAck(sequence, self.pool); err == nil {
				wm.Ref()
				if err := wm.WriteMessage(self.conn, self.peer); err == nil {
					//logrus.Infof("{@%d} ->", sequence)
				} else {
					logrus.Errorf("{@%d} -> (%v)", sequence, err)
					self.txErrors <- errors.Wrap(err, "write ack")
				}
				wm.Unref()

			} else {
				logrus.Errorf("{@%d} -> (%v)", sequence, err)
			}
		}
	}
}

func (self *txWindow) monitor() {
	logrus.Infof("started")
	defer logrus.Warnf("exited")

	for {
		var timeout time.Duration

		self.lock.Lock()
		for len(self.txMonitor.waiting) < 1 {
			self.txMonitor.ready.Wait()
		}
		self.txMonitor.monitoring = self.txMonitor.waiting[0]
		timeout = time.Until(self.txMonitor.monitoring.timeout)
		self.txMonitor.cancelled = false
		self.lock.Unlock()

		time.Sleep(timeout)

		self.lock.Lock()
		if !self.txMonitor.cancelled {
			logrus.Warnf("[!%d] ->", self.txMonitor.monitoring.wm.Sequence)
			self.txQueue <- self.txMonitor.monitoring.wm
			self.txMonitor.monitoring.timeout = time.Now().Add(retransmissionDelayMs * time.Millisecond)
		}
		self.lock.Unlock()
	}
}

func (self *txWindow) addMonitor(wm *wb.WireMessage) {
	timeout := time.Now().Add(retransmissionDelayMs * time.Millisecond)
	self.txMonitor.waiting = append(self.txMonitor.waiting, &txMonitored{timeout: timeout, wm: wm})
	self.txMonitor.ready.Signal()
}

func (self *txWindow) cancelMonitor(wm *wb.WireMessage) {
	i := -1
	for j, monitor := range self.txMonitor.waiting {
		if monitor.wm == wm {
			i = j
			break
		}
	}
	if i > -1 {
		self.txMonitor.waiting = append(self.txMonitor.waiting[:i], self.txMonitor.waiting[i+1:]...)
	}
	if self.txMonitor.monitoring != nil && self.txMonitor.monitoring.wm == wm {
		self.txMonitor.cancelled = true
	}
}
