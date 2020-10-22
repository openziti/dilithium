package westworld3

import (
	"github.com/emirpasic/gods/trees/btree"
	"github.com/emirpasic/gods/utils"
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
		monitor:      nil,
		closeWaitSeq: -1,
		closed:       false,
		conn:         conn,
		peer:         peer,
		profile:      profile,
		ii:           ii,
	}
	p.ready = sync.NewCond(p.lock)
	p.monitor.ready = sync.NewCond(p.lock)
	// go p.retxMonitor
	// go p.watchdog
	return p
}
