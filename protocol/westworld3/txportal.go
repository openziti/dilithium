package westworld3

import (
	"github.com/emirpasic/gods/trees/btree"
	"github.com/emirpasic/gods/utils"
	"github.com/michaelquigley/dilithium/util"
	"github.com/pkg/errors"
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
		//segmentSz := int(math.Min(float64(remaining), float64(self.profile.MaxSegmentSz)))
	}

	return n, nil
}
