package westworld3

import (
	"github.com/emirpasic/gods/trees/btree"
	"github.com/emirpasic/gods/utils"
	"github.com/openziti/dilithium/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"io"
	"math"
	"net"
	"sync"
	"time"
)

type txPortal struct {
	lock              *sync.Mutex
	tree              *btree.Tree
	capacity          int
	ready             *sync.Cond
	txPortalSz        int
	rxPortalSz        int
	successCt         int
	successAccum      int
	dupAckCt          int
	retxCt            int
	startRetxScale    float64
	lastRetxScaleIncr time.Time
	lastRetxScaleDecr time.Time
	lastRttProbe      time.Time
	monitor           *retxMonitor
	closer            *closer
	closeSent         bool
	closed            bool
	conn              *net.UDPConn
	peer              *net.UDPAddr
	pool              *pool
	profile           *Profile
	ii                InstrumentInstance
}

func newTxPortal(conn *net.UDPConn, peer *net.UDPAddr, closer *closer, profile *Profile, pool *pool, ii InstrumentInstance) *txPortal {
	p := &txPortal{
		lock:              new(sync.Mutex),
		tree:              btree.NewWith(profile.TxPortalTreeLen, utils.Int32Comparator),
		capacity:          profile.TxPortalStartSz,
		startRetxScale:    profile.RetxScale,
		lastRetxScaleIncr: time.Now(),
		lastRetxScaleDecr: time.Now(),
		rxPortalSz:        -1,
		closer:            closer,
		closed:            false,
		conn:              conn,
		peer:              peer,
		pool:              pool,
		profile:           profile,
		ii:                ii,
	}
	p.ready = sync.NewCond(p.lock)
	p.monitor = newRetxMonitor(profile, conn, peer, p.lock, p.ii)
	p.monitor.setRetxF(p.retx)
	return p
}

func (self *txPortal) tx(p []byte, seq *util.Sequence) (n int, err error) {
	self.lock.Lock()
	defer self.lock.Unlock()

	if self.closed {
		return -1, io.EOF
	}

	remaining := len(p)
	n = 0
	for remaining > 0 {
		segmentSz := int(math.Min(float64(remaining), float64(self.profile.MaxSegmentSz)))

		var rtt *uint16
		if time.Since(self.lastRttProbe).Milliseconds() > int64(self.profile.RttProbeMs) {
			now := time.Now()
			rtt = new(uint16)
			*rtt = uint16(now.UnixNano() / int64(time.Millisecond))
			segmentSz -= 2
			self.lastRttProbe = now
		}

		for self.availableCapacity(segmentSz) < 0 {
			self.ready.Wait()
		}

		wm, err := newData(seq.Next(), rtt, p[n:n+segmentSz], self.pool)
		if err != nil {
			return 0, errors.Wrap(err, "new data")
		}
		self.tree.Put(wm.seq, wm)
		self.txPortalSz += segmentSz
		self.ii.TxPortalSzChanged(self.peer, self.txPortalSz)

		if err := writeWireMessage(wm, self.conn, self.peer); err != nil {
			return 0, errors.Wrap(err, "tx")
		}
		self.ii.WireMessageTx(self.peer, wm)

		self.monitor.add(wm)

		n += segmentSz
		remaining -= segmentSz
	}

	return n, nil
}

func (self *txPortal) ack(acks []ack) error {
	self.lock.Lock()
	defer self.lock.Unlock()

	lastTxPortalSz := self.txPortalSz
	for _, ack := range acks {
		for seq := ack.start; seq <= ack.end; seq++ {
			if v, found := self.tree.Get(seq); found {
				wm := v.(*wireMessage)
				self.monitor.remove(wm)
				self.tree.Remove(seq)
				switch wm.messageType() {
				case DATA:
					sz, err := wm.asDataSize()
					if err != nil {
						return errors.Wrap(err, "internal tree error")
					}
					self.txPortalSz -= int(sz)
					self.ii.TxPortalSzChanged(self.peer, self.txPortalSz)
					self.successfulAck(int(sz))

				case CLOSE:
					self.successfulAck(0)

				default:
					logrus.Warnf("acked suspicious message type in tree [%d]", wm.messageType())
				}
				wm.buffer.unref()

			} else {
				self.duplicateAck(seq)
			}
		}
	}

	if self.txPortalSz != lastTxPortalSz {
		self.ii.TxPortalSzChanged(self.peer, self.txPortalSz)
	}

	if time.Since(self.lastRetxScaleDecr).Milliseconds() > int64(self.profile.RetxEvaluationMs) {
		self.profile.RetxScale -= self.profile.RetxEvaluationScaleDecr
		if self.profile.RetxScale < self.profile.RetxScaleFloor {
			self.profile.RetxScale = self.profile.RetxScaleFloor
		}
		self.ii.NewRetxScale(self.peer, self.profile.RetxScale)
		self.lastRetxScaleDecr = time.Now()
	}

	self.ready.Broadcast()
	return nil
}

func (self *txPortal) updateRxPortalSz(rxPortalSz int) {
	self.lock.Lock()
	defer self.lock.Unlock()
	self.rxPortalSz = rxPortalSz
	self.ready.Broadcast()
	self.ii.TxPortalRxSzChanged(self.peer, rxPortalSz)
}

func (self *txPortal) rtt(probeTs uint16) {
	now := time.Now().UnixNano()
	self.lock.Lock()
	clockTs := uint16(now / int64(time.Millisecond))
	rttMs := clockTs - probeTs
	self.monitor.updateRttMs(rttMs)
	self.lock.Unlock()
}

func (self *txPortal) sendClose(seq *util.Sequence) error {
	self.lock.Lock()
	defer self.lock.Unlock()

	if !self.closeSent {
		wm, err := newClose(seq.Next(), self.pool)
		if err != nil {
			return errors.Wrap(err, "close")
		}
		self.tree.Put(wm.seq, wm)
		self.monitor.add(wm)

		if err := writeWireMessage(wm, self.conn, self.peer); err != nil {
			return errors.Wrap(err, "tx close")
		}
		self.closer.txCloseSeqIn <- wm.seq
		self.ii.WireMessageTx(self.peer, wm)

		self.closeSent = true
	}

	return nil
}

func (self *txPortal) close() {
	self.closed = true
	self.monitor.closed = true
}

func (self *txPortal) successfulAck(sz int) {
	self.successCt++
	self.successAccum += sz
	if self.successCt == self.profile.TxPortalIncreaseThresh {
		newCapacity := self.capacity + int(float64(self.successAccum)*self.profile.TxPortalIncreaseScale)
		self.updatePortalCapacity(newCapacity)
		self.successCt = 0
		self.successAccum = 0
	}
}

func (self *txPortal) duplicateAck(seq int32) {
	self.dupAckCt++
	self.successCt = 0
	if self.dupAckCt >= self.profile.TxPortalDupAckThresh {
		newCapacity := int(float64(self.capacity) * self.profile.TxPortalDupAckCapacityScale)

		// #93: Self-Adjusting retxMs
		if time.Since(self.lastRetxScaleIncr).Milliseconds() > int64(self.profile.RetxEvaluationMs) {
			self.profile.RetxScale += self.profile.RetxEvaluationScaleIncr
			self.lastRetxScaleIncr = time.Now()
			self.ii.NewRetxScale(self.peer, self.profile.RetxScale)
		}

		self.updatePortalCapacity(newCapacity)
		self.dupAckCt = 0
		self.successAccum = int(float64(self.successAccum) * self.profile.TxPortalDupAckSuccessScale)
	}
	self.ii.DuplicateAck(self.peer, seq)
}

func (self *txPortal) retx() {
	self.retxCt++
	self.successCt = 0
	if self.retxCt >= self.profile.TxPortalRetxThresh {
		newCapacity := int(float64(self.capacity) * self.profile.TxPortalRetxCapacityScale)
		self.updatePortalCapacity(newCapacity)
		self.retxCt = 0
		self.successAccum = int(float64(self.successAccum) * self.profile.TxPortalRetxSuccessScale)
	}
}

func (self *txPortal) updatePortalCapacity(newCapacity int) {
	oldCapacity := self.capacity
	self.capacity = newCapacity
	if self.capacity < self.profile.TxPortalMinSz {
		self.capacity = self.profile.TxPortalMinSz
	}
	if self.capacity > self.profile.TxPortalMaxSz {
		self.capacity = self.profile.TxPortalMaxSz
	}
	if self.capacity != oldCapacity {
		self.ii.TxPortalCapacityChanged(self.peer, self.capacity)
	}
}

func (self *txPortal) availableCapacity(segmentSz int) int {
	txPortalCapacity := float64(self.capacity - int(float64(self.rxPortalSz)*self.profile.TxPortalRxSzPressureScale) - (self.txPortalSz + segmentSz))
	rxPortalCapacity := float64(self.capacity - (self.rxPortalSz + segmentSz))
	return int(math.Min(txPortalCapacity, rxPortalCapacity))
}
