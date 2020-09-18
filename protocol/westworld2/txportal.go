package westworld2

import (
	"github.com/emirpasic/gods/trees/btree"
	"github.com/emirpasic/gods/utils"
	"github.com/michaelquigley/dilithium/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"math"
	"net"
	"sort"
	"sync"
	"time"
)

type txPortal struct {
	lock       *sync.Mutex
	tree       *btree.Tree
	capacity   int
	succCt     int
	succAccum  int
	dupackCt   int
	retxCt     int
	ready      *sync.Cond
	monitor    *retxMonitor
	closeWait  int32
	closed     bool
	lastRtt    time.Time
	rttw       []int
	retxMs     int
	rxPortalSz int32
	conn       *net.UDPConn
	peer       *net.UDPAddr
	pool       *pool
	config     *Config
	count      int
}

type retxMonitor struct {
	waiting []*retxSubject
	ready   *sync.Cond
}

type retxSubject struct {
	deadline time.Time
	wm       *wireMessage
}

func newTxPortal(conn *net.UDPConn, peer *net.UDPAddr, config *Config) *txPortal {
	txp := &txPortal{
		lock:       new(sync.Mutex),
		tree:       btree.NewWith(config.treeLen, utils.Int32Comparator),
		capacity:   config.txPortalStartSz,
		monitor:    &retxMonitor{},
		closeWait:  -1,
		closed:     false,
		retxMs:     config.retxStartMs,
		rxPortalSz: -1,
		conn:       conn,
		peer:       peer,
		pool:       newPool("txPortal", config),
		config:     config,
		count:      0,
	}
	txp.ready = sync.NewCond(txp.lock)
	txp.monitor.ready = sync.NewCond(txp.lock)
	go txp.runMonitor()
	return txp
}

func (self *txPortal) tx(p []byte, seq *util.Sequence) (n int, err error) {
	self.lock.Lock()
	defer self.lock.Unlock()

	if self.closeWait != -1 || self.closed {
		return 0, errors.New("closed")
	}

	remaining := len(p)
	n = 0
	for remaining > 0 {
		self.count++
		if self.count%treeReportCt == 0 {
			logrus.Infof("tree.Size = %d, rxPortalSz = %d", self.tree.Size(), self.rxPortalSz)
		}

		sz := int(math.Min(float64(remaining), float64(self.config.maxSegmentSz)))
		wm := newData(seq.Next(), p[n:n+sz], self.pool)

		for self.capacity < sz {
			self.ready.Wait()
		}

		self.tree.Put(wm.seq, wm)
		self.capacity -= sz

		if time.Since(self.lastRtt).Milliseconds() > int64(self.config.rttProbeMs) {
			rttWm, err := wm.clone()
			if err != nil {
				return n, err
			}
			rttWm.writeRtt(time.Now().UnixNano())
			if err = writeWireMessage(rttWm, self.conn, self.peer); err != nil {
				rttWm.buffer.unref()
				return 0, errors.Wrap(err, "rttTx")
			}
			if self.config.i != nil {
				self.config.i.wireMessageTx(self.peer, rttWm)
			}
			rttWm.buffer.unref()
			self.lastRtt = time.Now()

		} else {
			if err = writeWireMessage(wm, self.conn, self.peer); err != nil {
				return 0, errors.Wrap(err, "tx")
			}
			if self.config.i != nil {
				self.config.i.wireMessageTx(self.peer, wm)
			}
		}

		self.addMonitor(wm)

		n += sz
		remaining -= sz
	}

	return n, nil
}

func (self *txPortal) close(seq *util.Sequence) error {
	self.lock.Lock()
	defer self.lock.Unlock()

	if !self.closed {
		wm := newClose(seq.Next(), self.pool)
		self.closeWait = wm.seq
		self.tree.Put(wm.seq, wm)
		self.addMonitor(wm)

		if err := writeWireMessage(wm, self.conn, self.peer); err != nil {
			return errors.Wrap(err, "tx close")
		}
		if self.config.i != nil {
			self.config.i.wireMessageTx(self.peer, wm)
		}
	}

	return nil
}

func (self *txPortal) ack(sequence, rxPortalSz int32) {
	self.lock.Lock()
	defer self.lock.Unlock()

	if rxPortalSz > -1 {
		self.rxPortalSz = rxPortalSz
	}

	if v, found := self.tree.Get(sequence); found {
		wm := v.(*wireMessage)
		self.cancelMonitor(wm)
		self.tree.Remove(sequence)
		sz := len(wm.data)
		self.capacity += sz
		wm.buffer.unref()

		self.portalSuccessfulAck(sz)

		if wm.seq == self.closeWait {
			self.closed = true
		}

		self.ready.Signal()

	} else {
		self.portalDuplicateAck(sequence)
	}
}

func (self *txPortal) rtt(ts int64) {
	self.lock.Lock()
	defer self.lock.Unlock()

	rttMs := int(time.Since(time.Unix(0, ts)).Milliseconds())
	self.rttw = append(self.rttw, rttMs)
	if len(self.rttw) > self.config.rttProbeAvgCt {
		self.rttw = self.rttw[1:]
	}
	i := 0
	accum := 0
	for ; i < len(self.rttw); i++ {
		accum += self.rttw[i]
	}
	if i > 0 {
		self.retxMs = accum / i
	}
	if self.retxMs < 5 {
		self.retxMs = 5
	}

	if self.config.i != nil {
		self.config.i.newRetxMs(self.peer, self.retxMs)
	}
}

func (self *txPortal) runMonitor() {
	logrus.Info("started")
	defer logrus.Warn("exited")

	for {
		var headline time.Time
		var timeout time.Duration

		self.lock.Lock()
		{
			if self.closed {
				if self.config.i != nil {
					self.config.i.closed(self.peer)
				}
				return
			}

			for len(self.monitor.waiting) < 1 {
				self.monitor.ready.Wait()
			}
			headline = self.monitor.waiting[0].deadline
			timeout = time.Until(headline)
		}
		self.lock.Unlock()

		time.Sleep(timeout)

		self.lock.Lock()
		{
			if len(self.monitor.waiting) > 0 {
				i := 0
				x := len(self.monitor.waiting)
				for ; i < x; i++ {
					delta := self.monitor.waiting[i].deadline.Sub(headline).Milliseconds()
					if delta <= 2 {
						if self.config.i != nil {
							self.config.i.wireMessageRetx(self.peer, self.monitor.waiting[i].wm)
						}
						if err := writeWireMessage(self.monitor.waiting[i].wm, self.conn, self.peer); err != nil {
							logrus.Errorf("retx (%v)", err)
						} else {
							if self.config.i != nil {
								self.config.i.wireMessageRetx(self.peer, self.monitor.waiting[i].wm)
							}
						}
						self.portalRetx()

						self.monitor.waiting[i].deadline = time.Now().Add(time.Duration(self.retxMs+self.config.retxAddMs) * time.Millisecond)
						self.monitor.waiting = append(self.monitor.waiting, self.monitor.waiting[i])
					} else {
						break
					}
				}
				self.monitor.waiting = self.monitor.waiting[i:]
				sort.Slice(self.monitor.waiting, func(i, j int) bool {
					return self.monitor.waiting[i].deadline.Before(self.monitor.waiting[j].deadline)
				})
			}
		}
		self.lock.Unlock()
	}
}

func (self *txPortal) addMonitor(wm *wireMessage) {
	self.monitor.waiting = append(self.monitor.waiting, &retxSubject{time.Now().Add(time.Duration(self.retxMs+self.config.retxAddMs) * time.Millisecond), wm})
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
		self.monitor.waiting = append(self.monitor.waiting[:i], self.monitor.waiting[i+1:]...)
	} else {
		panic("cancelled non-existent monitor")
	}
}

func (self *txPortal) portalSuccessfulAck(sz int) {
	self.succCt++
	self.succAccum += sz
	if self.succCt == self.config.txPortalIncreaseCt {
		self.updatePortalSz(self.capacity + int(float64(self.succAccum)*self.config.txPortalIncreaseFrac))
		self.succCt = 0
		self.succAccum = 0
	}
}

func (self *txPortal) portalDuplicateAck(sequence int32) {
	self.dupackCt++
	self.succCt = 0
	if self.dupackCt == self.config.txPortalDupAckCt {
		self.updatePortalSz(int(float64(self.capacity) * self.config.txPortalDupAckFrac))
		self.dupackCt = 0
		self.succAccum = 0
	}
	if self.config.i != nil {
		self.config.i.duplicateAck(self.peer, sequence)
	}
}

func (self *txPortal) portalRetx() {
	self.retxCt++
	if self.retxCt == self.config.txPortalRetxCt {
		self.updatePortalSz(int(float64(self.capacity) * self.config.txPortalRetxFrac))
		self.retxCt = 0
		self.succAccum = 0
	}
}

func (self *txPortal) updatePortalSz(newCapacity int) {
	self.capacity = newCapacity
	if self.capacity < self.config.txPortalMinSz {
		self.capacity = self.config.txPortalMinSz
	}
	if self.config.txPortalMaxSz > 0 && self.capacity > self.config.txPortalMaxSz {
		self.capacity = self.config.txPortalMaxSz
	}
	if self.config.i != nil {
		self.config.i.portalCapacitySz(self.peer, self.capacity)
	}
}
