package westworld3

import (
	"github.com/emirpasic/gods/trees/btree"
	"github.com/emirpasic/gods/utils"
	"github.com/michaelquigley/dilithium/util"
	"github.com/pkg/errors"
	"math"
	"net"
	"sync"
	"time"
)

type txPortal struct {
	lock         *sync.Mutex
	tree         *btree.Tree
	capacity     int
	ready        *sync.Cond
	txPortalSz   int
	rxPortalSz   int
	successCt    int
	successAccum int
	dupAckCt     int
	retxCt       int
	rttProbes    []int
	retxMs       int
	lastRttProbe time.Time
	monitor      *retxMonitor
	closeWaitSeq int32
	closed       bool
	conn         *net.UDPConn
	peer         *net.UDPAddr
	pool *pool
	profile      *Profile
	ii           InstrumentInstance
}

func newTxPortal(conn *net.UDPConn, peer *net.UDPAddr, profile *Profile, ii InstrumentInstance) *txPortal {
	p := &txPortal{
		lock:         new(sync.Mutex),
		tree:         btree.NewWith(profile.TxPortalTreeLen, utils.Int32Comparator),
		capacity:     profile.TxPortalStartSz,
		rxPortalSz:   -1,
		retxMs:       profile.RetxStartMs,
		closeWaitSeq: -1,
		closed:       false,
		conn:         conn,
		peer:         peer,
		profile:      profile,
		ii:           ii,
	}
	p.ready = sync.NewCond(p.lock)
	p.monitor = newRetxMonitor(profile, conn, peer, p.lock)
	// go p.watchdog
	return p
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
		segmentSz := int(math.Min(float64(remaining), float64(self.profile.MaxSegmentSz)))

		var rtt *uint16
		if time.Since(self.lastRttProbe).Milliseconds() > int64(self.profile.RttProbeMs) {
			rtt = new(uint16)
			*rtt = uint16(time.Now().UnixNano() / int64(time.Millisecond))
			segmentSz -= 2
		}

		for math.Min(float64(self.capacity-(self.txPortalSz+segmentSz)), float64(self.capacity-self.rxPortalSz)) < 0 {
			self.ready.Wait()
		}

		wm, err := newData(seq.Next(), rtt, p[n:n+segmentSz], self.pool)
		if err != nil {
			return 0, errors.Wrap(err, "new data")
		}
		self.tree.Put(wm.seq, wm)
		self.txPortalSz += segmentSz

		if err := writeWireMessage(wm, self.conn, self.peer); err != nil {
			return 0, errors.Wrap(err, "tx")
		}

		self.monitor.monitor(wm)

		n += segmentSz
		remaining -= segmentSz
	}

	return n, nil
}

func (self *txPortal) ack(peer *net.UDPAddr, seq int32, rxPortalSz int) error {
	self.lock.Lock()
	defer self.lock.Unlock()

	if rxPortalSz > -1 {
		self.rxPortalSz = rxPortalSz
	}

	if v, found := self.tree.Get(seq); found {
		wm := v.(*wireMessage)
		self.monitor.cancel(wm)
		self.tree.Remove(seq)
		sz, err := wm.asDataSize()
		if err != nil {
			return errors.Wrap(err, "not data")
		}
		self.txPortalSz -= int(sz)
		wm.buffer.unref()

		if wm.seq == self.closeWaitSeq {
			self.closed = true
		}

		self.ready.Broadcast()

	} else {
		// dupack
	}

	return nil
}

func (self *txPortal) close(seq *util.Sequence) error {
	self.lock.Lock()
	defer self.lock.Unlock()

	if !self.closed {
		wm, err := newClose(seq.Next(), self.pool)
		if err != nil {
			return errors.Wrap(err, "close")
		}
		self.closeWaitSeq = wm.seq
		self.tree.Put(wm.seq, wm)
		self.monitor.monitor(wm)

		if err := writeWireMessage(wm, self.conn, self.peer); err != nil {
			return errors.Wrap(err, "tx close")
		}
	}

	return nil
}