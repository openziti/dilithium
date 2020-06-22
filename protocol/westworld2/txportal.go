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
	lock     *sync.Mutex
	tree     *btree.Tree
	capacity int
	ready    *sync.Cond
	monitor  *retxMonitor
	lastRtt  time.Time
	rttw     []int
	retxMs   int
	conn     *net.UDPConn
	peer     *net.UDPAddr
	pool     *pool
	config   *Config
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

func newTxPortal(conn *net.UDPConn, peer *net.UDPAddr, config *Config) *txPortal {
	txp := &txPortal{
		lock:     new(sync.Mutex),
		tree:     btree.NewWith(config.treeLen, utils.Int32Comparator),
		capacity: config.portalStartSz,
		monitor:  &retxMonitor{},
		retxMs:   config.retxTimeoutMs,
		conn:     conn,
		peer:     peer,
		pool:     newPool("txPortal", config),
		config:   config,
	}
	txp.ready = sync.NewCond(txp.lock)
	txp.monitor.ready = sync.NewCond(txp.lock)
	go txp.runMonitor()
	return txp
}

func (self *txPortal) tx(p []byte, seq *util.Sequence) (n int, err error) {
	self.lock.Lock()
	defer self.lock.Unlock()

	remaining := len(p)
	n = 0
	for remaining > 0 {
		sz := int(math.Min(float64(remaining), float64(self.config.maxSegmentSz)))
		wm := newData(seq.Next(), p[n:n+sz], self.pool)

		for self.capacity < sz {
			self.ready.Wait()
		}

		self.tree.Put(wm.seq, wm)
		self.capacity -= sz
		self.addMonitor(wm)

		if time.Since(self.lastRtt).Milliseconds() > 1000 {
			rttWm, err := wm.clone()
			if err != nil {
				return n, err
			}
			rttWm.writeRtt(time.Now().UnixNano())
			if err = writeWireMessage(rttWm, self.conn, self.peer, self.config.i); err != nil {
				rttWm.buffer.unref()
				return 0, errors.Wrap(err, "rttTx")
			}
			rttWm.buffer.unref()
			self.lastRtt = time.Now()

		} else {
			if err = writeWireMessage(wm, self.conn, self.peer, self.config.i); err != nil {
				return 0, errors.Wrap(err, "tx")
			}
		}

		n += sz
		remaining -= sz
	}

	return n, nil
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
		if self.config.i != nil {
			self.config.i.duplicateAck(self.peer, sequence)
		}
	}
}

func (self *txPortal) rtt(ts int64) {
	self.lock.Lock()
	defer self.lock.Unlock()

	logrus.Infof("ts in = [%d]", ts)
	rttMs := int(time.Since(time.Unix(0, ts)).Milliseconds())
	self.rttw = append(self.rttw, rttMs)
	if len(self.rttw) > 8 {
		self.rttw = self.rttw[1:]
	}
	i := 0
	accum := 0
	for ; i < len(self.rttw); i++ {
		accum += self.rttw[i]
	}
	if i > 0 {
		self.retxMs = accum / i
		self.retxMs += int(float32(self.retxMs) * 0.1)
	}
	if self.retxMs < 5 {
		self.retxMs = 5
	}
	logrus.Infof("smoothed retxMs = [%d]", self.retxMs)
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
			if self.config.i != nil {
				self.config.i.wireMessageRetx(self.peer, self.monitor.head.wm)
			}
			if err := writeWireMessage(self.monitor.head.wm, self.conn, self.peer, self.config.i); err != nil {
				logrus.Errorf("retx (%v)", err)
			}
			self.monitor.head.deadline = time.Now().Add(time.Duration(self.retxMs) * time.Millisecond)
			sort.Slice(self.monitor.waiting, func(i, j int) bool {
				return self.monitor.waiting[i].deadline.Before(self.monitor.waiting[j].deadline)
			})
		}
		self.lock.Unlock()
	}
}

func (self *txPortal) addMonitor(wm *wireMessage) {
	wm.buffer.ref()
	self.monitor.waiting = append(self.monitor.waiting, &retxSubject{time.Now().Add(time.Duration(self.retxMs) * time.Millisecond), wm})
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
