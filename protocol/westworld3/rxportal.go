package westworld3

import (
	"bytes"
	"github.com/emirpasic/gods/trees/btree"
	"github.com/emirpasic/gods/utils"
	"github.com/michaelquigley/dilithium/util"
	"net"
	"sync"
)

type rxPortal struct {
	tree       *btree.Tree
	accepted   int32
	rxs        chan *wireMessage
	reads      chan *rxRead
	readBuffer *bytes.Buffer
	rxPortalSz int
	readPool   *sync.Pool
	ackPool    *pool
	conn       *net.UDPConn
	peer       *net.UDPAddr
	txPortal   *txPortal
	seq        *util.Sequence
	profile    *Profile
	closed     bool
}

type rxRead struct {
	buf []byte
	sz  int
	eof bool
}

func newRxPortal(conn *net.UDPConn, peer *net.UDPAddr, txPortal *txPortal, seq *util.Sequence, profile *Profile) *rxPortal {
	rx := &rxPortal{
		tree:       btree.NewWith(profile.RxPortalTreeLen, utils.Int32Comparator),
		accepted:   -1,
		rxs:        make(chan *wireMessage),
		reads:      make(chan *rxRead, profile.ReadsQueueLen),
		readBuffer: new(bytes.Buffer),
		readPool:   new(sync.Pool),
		ackPool:    newPool("ackPool", uint32(profile.PoolBufferSz), nil),
		conn:       conn,
		peer:       peer,
		txPortal:   txPortal,
		seq:        seq,
		profile:    profile,
	}
	rx.readPool.New = func() interface{} {
		return make([]byte, profile.PoolBufferSz)
	}
	go rx.run()
	return rx
}

func (self *rxPortal) run() {
}
