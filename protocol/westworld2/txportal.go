package westworld2

import (
	"fmt"
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
	lock         *sync.Mutex
	tree         *btree.Tree
	capacity     int
	txPortalSz   int
	rxPortalSz   int
	succCt       int
	succAccum    int
	dupAckCt     int
	dupAckAllCt  int
	retxCt       int
	retxAllCt    int
	ready        *sync.Cond
	monitor      *retxMonitor
	lastRtt      time.Time
	rttw         []int
	retxMs       int
	closeWaitSeq int32
	closed       bool
	conn         *net.UDPConn
	peer         *net.UDPAddr
	pool         *pool
	config       *Config
	ii           InstrumentInstance
	txCount      int64
}

type retxMonitor struct {
	waiting []*retxSubject
	ready   *sync.Cond
}

type retxSubject struct {
	deadline time.Time
	wm       *wireMessage
}

func newTxPortal(conn *net.UDPConn, peer *net.UDPAddr, config *Config, ii InstrumentInstance) *txPortal {
	txp := &txPortal{
		lock:         new(sync.Mutex),
		tree:         btree.NewWith(config.treeLen, utils.Int32Comparator),
		capacity:     config.txPortalStartSz,
		rxPortalSz:   -1,
		retxMs:       config.retxStartMs,
		monitor:      &retxMonitor{},
		closeWaitSeq: -1,
		closed:       false,
		conn:         conn,
		peer:         peer,
		pool:         newPool("txPortal", ii),
		config:       config,
		ii:           ii,
	}
	txp.ready = sync.NewCond(txp.lock)
	txp.monitor.ready = sync.NewCond(txp.lock)
	go txp.retxMonitor()
	go txp.watchdog()
	return txp
}

func (self *txPortal) tx(p []byte, seq *util.Sequence) (n int, err error) {
	self.lock.Lock()
	defer self.lock.Unlock()

	if self.closeWaitSeq != -1 || self.closed {
		return 0, errors.New("closed")
	}

	remaining := len(p)
	n = 0
	for remaining > 0 {
		sz := int(math.Min(float64(remaining), float64(self.config.maxSegmentSz)))
		wm := newData(seq.Next(), p[n:n+sz], self.pool)

		for math.Min(float64(self.capacity-int(self.txPortalSz+sz)), float64(self.capacity-int(self.rxPortalSz))) < 0 {
			self.ready.Wait()
		}

		self.tree.Put(wm.seq, wm)
		self.txPortalSz += sz

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
			if self.ii != nil {
				self.ii.wireMessageTx(self.peer, rttWm)
			}
			rttWm.buffer.unref()
			self.lastRtt = time.Now()
			self.txCount++

		} else {
			if err = writeWireMessage(wm, self.conn, self.peer); err != nil {
				return 0, errors.Wrap(err, "tx")
			}
			if self.ii != nil {
				self.ii.wireMessageTx(self.peer, wm)
			}
			self.txCount++
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
		self.closeWaitSeq = wm.seq
		self.tree.Put(wm.seq, wm)
		self.addMonitor(wm)

		if err := writeWireMessage(wm, self.conn, self.peer); err != nil {
			return errors.Wrap(err, "tx close")
		}
		if self.ii != nil {
			self.ii.wireMessageTx(self.peer, wm)
		}
	}

	return nil
}

func (self *txPortal) ack(peer *net.UDPAddr, sequence int32, rxPortalSz int) {
	self.lock.Lock()
	defer self.lock.Unlock()

	if rxPortalSz > -1 {
		self.rxPortalSz = rxPortalSz
		if self.ii != nil {
			self.ii.txPortalRxPortalSzChanged(peer, int(rxPortalSz))
		}
	}

	if v, found := self.tree.Get(sequence); found {
		wm := v.(*wireMessage)
		self.cancelMonitor(wm)
		self.tree.Remove(sequence)
		sz := len(wm.data)
		self.txPortalSz -= sz
		if self.ii != nil {
			self.ii.txPortalSzChanged(peer, self.txPortalSz)
		}
		wm.buffer.unref()

		self.portalSuccessfulAck(sz)

		if wm.seq == self.closeWaitSeq {
			self.closed = true
		}

		self.ready.Broadcast()

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

	if self.ii != nil {
		self.ii.newRetxMs(self.peer, self.retxMs)
	}
}

func (self *txPortal) watchdog() {
	for {
		last := self.txCount
		time.Sleep(1 * time.Second)
		logrus.Infof("\n%s\n%s\n%s\n%s\n\n",
			fmt.Sprintf("txCount = %d (%d)", self.txCount, self.txCount-last),
			fmt.Sprintf("capacity = %d, txPortalSz = %d, rxPortalSz = %d", self.capacity, self.txPortalSz, self.rxPortalSz),
			fmt.Sprintf("retx = %d, dupAck = %d", self.retxAllCt, self.dupAckAllCt),
			fmt.Sprintf("retxMs = %d, monitoring = %d", self.retxMs, len(self.monitor.waiting)),
		)
		self.ready.Broadcast()
		self.monitor.ready.Broadcast()
	}
}

func (self *txPortal) retxMonitor() {
	logrus.Info("started")
	defer logrus.Warn("exited")

	for {
		var headline time.Time
		var timeout time.Duration

		self.lock.Lock()
		{
			if self.closed {
				if self.ii != nil {
					self.ii.closed(self.peer)
				}
				return
			}

			for len(self.monitor.waiting) < 1 {
				dup := newData(-33, nil, self.pool)
				if err := writeWireMessage(dup, self.conn, self.peer); err != nil {
					logrus.Errorf("error provoking peer (%v)", err)
				}
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
						if self.ii != nil {
							self.ii.wireMessageRetx(self.peer, self.monitor.waiting[i].wm)
						}
						if err := writeWireMessage(self.monitor.waiting[i].wm, self.conn, self.peer); err != nil {
							logrus.Errorf("retx (%v)", err)
						} else {
							if self.ii != nil {
								self.ii.wireMessageRetx(self.peer, self.monitor.waiting[i].wm)
							}
							self.txCount++
						}
						self.portalRetx()

						self.monitor.waiting[i].deadline = time.Now().Add(time.Duration((self.retxMs*2)+self.config.retxAddMs) * time.Millisecond)
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
	self.monitor.waiting = append(self.monitor.waiting, &retxSubject{time.Now().Add(time.Duration((self.retxMs*2)+self.config.retxAddMs) * time.Millisecond), wm})
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
	self.dupAckCt++
	self.dupAckAllCt++
	self.succCt = 0
	if self.dupAckCt == self.config.txPortalDupAckCt {
		self.updatePortalSz(int(float64(self.capacity) * self.config.txPortalDupAckFrac))
		self.dupAckCt = 0
		self.succAccum = 0
	}
	if self.ii != nil {
		self.ii.duplicateAck(self.peer, sequence)
	}
}

func (self *txPortal) portalRetx() {
	self.retxCt++
	self.retxAllCt++
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
	if self.ii != nil {
		self.ii.txPortalCapacityChanged(self.peer, self.capacity)
	}
}
