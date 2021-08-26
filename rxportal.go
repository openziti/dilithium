package dilithium

import (
	"bytes"
	"github.com/emirpasic/gods/trees/btree"
	"github.com/emirpasic/gods/utils"
	"github.com/openziti/dilithium/util"
	"sync"
)

type RxPortal struct {
	transport    Transport
	tree         *btree.Tree
	accepted     int32
	rxs          chan *WireMessage
	reads        chan *RxRead
	readBuffer   *bytes.Buffer
	rxPortalSize int
	readPool     *sync.Pool
	ackPool      *Pool
	txp          *TxPortal
	seq          *util.Sequence
	closer       *Closer
}

type RxRead struct {
	Buf  []byte
	Size int
	Eof  bool
}

func NewRxPortal(transport Transport, txp *TxPortal, seq *util.Sequence, closer *Closer) *RxPortal {
	rxp := &RxPortal{
		transport:  transport,
		tree:       btree.NewWith(txp.alg.Profile().MaxTreeSize, utils.Int32Comparator),
		accepted:   -1,
		rxs:        make(chan *WireMessage),
		reads:      make(chan *RxRead, txp.alg.Profile().ReadsQueueSize),
		readBuffer: new(bytes.Buffer),
		readPool:   new(sync.Pool),
		ackPool:    NewPool("ackPool", uint32(txp.alg.Profile().PoolBufferSize)),
		txp:        txp,
		seq:        seq,
		closer:     closer,
	}
	rxp.readPool.New = func() interface{} {
		return make([]byte, txp.alg.Profile().PoolBufferSize)
	}
	go rxp.run()
	return rxp
}

func (rxp *RxPortal) run() {
}