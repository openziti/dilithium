package conduit

import (
	"github.com/emirpasic/gods/trees/btree"
	"github.com/emirpasic/gods/utils"
	"github.com/sirupsen/logrus"
	"net"
	"sort"
	"sync"
	"time"
)

const startingWindowCapacity = 6
const retransmissionDelay = 10

type txWindow struct {
	tree             *btree.Tree
	lock             *sync.Mutex
	capacity         int
	capacityReady    *sync.Cond
	monitorWait      []*txMonitor
	monitoring       *txMonitor
	monitorCancelled bool
	monitorReady     *sync.Cond
	closed           bool
	conn             *net.UDPConn
	peer             *net.UDPAddr
}

type txMonitor struct {
	timeout time.Time
	retries int
	m       *message
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
	txw.monitorReady = sync.NewCond(txw.lock)
	go txw.retransmitter()
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
	self.addMonitor(m)
}

func (self *txWindow) ack(sequence int32) {
	self.lock.Lock()
	defer self.lock.Unlock()

	if m, found := self.tree.Get(sequence); found {
		self.cancelMonitor(m.(*message))
		self.tree.Remove(sequence)
		self.capacity++
		self.capacityReady.Broadcast()
	} else {
		logrus.Warnf("already acked [#%d]", sequence)
	}
}

func (self *txWindow) close() {
	self.closed = true
}

func (self *txWindow) retransmitter() {
	logrus.Infof("started")
	defer logrus.Warnf("exited")

	for {
		var timeout time.Duration
		{
			self.lock.Lock()

			for len(self.monitorWait) < 1 {
				if self.closed {
					self.lock.Unlock()
					return
				}
				self.monitorReady.Wait()
			}
			self.monitoring = self.monitorWait[0]
			timeout = time.Until(self.monitoring.timeout)
			self.monitorCancelled = false

			self.lock.Unlock()
		}

		time.Sleep(timeout)

		{
			self.lock.Lock()

			if !self.monitorCancelled {
				if data, err := self.monitoring.m.marshal(); err == nil {
					if n, err := self.conn.WriteTo(data, self.peer); err == nil {
						if n == len(data) {
							logrus.Warnf("[! %d] ->", self.monitoring.m.sequence)
						} else {
							logrus.Errorf("short retransmit")
						}
					} else {
						logrus.Errorf("error retransmitting (%v)", err)
					}
				} else {
					logrus.Errorf("error marshaling (%v)", err)
				}

				self.monitoring.timeout = time.Now().Add(retransmissionDelay * time.Millisecond)
				self.sortMonitorWait()

			} else {
				logrus.Debugf("[cancelled #%d]", self.monitoring.m.sequence)
			}

			self.lock.Unlock()
		}
	}
}

func (self *txWindow) addMonitor(m *message) {
	timeout := time.Now().Add(retransmissionDelay * time.Millisecond)
	self.monitorWait = append(self.monitorWait, &txMonitor{timeout: timeout, m: m})
	self.sortMonitorWait()
	self.monitorReady.Signal()
}

func (self *txWindow) cancelMonitor(m *message) {
	i := -1
	for j, monitor := range self.monitorWait {
		if monitor.m == m {
			i = j
			break
		}
	}
	if i > -1 {
		self.monitorWait = append(self.monitorWait[:i], self.monitorWait[i+1:]...)
	}
	if self.monitoring != nil && self.monitoring.m == m {
		self.monitorCancelled = true
	}
}

func (self *txWindow) sortMonitorWait() {
	sort.Slice(self.monitorWait, func(i, j int) bool {
		return self.monitorWait[i].timeout.Before(self.monitorWait[j].timeout)
	})
}
