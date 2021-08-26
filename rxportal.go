package dilithium

import (
	"bytes"
	"github.com/emirpasic/gods/trees/btree"
	"github.com/emirpasic/gods/utils"
	"github.com/openziti/dilithium/util"
	"github.com/pkg/errors"
	"io"
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
	closed       bool
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

func (rxp *RxPortal) Read(p []byte) (int, error) {
preread:
	for {
		select {
		case read, ok := <-rxp.reads:
			if !ok {
				return 0, io.EOF
			}
			if !read.Eof {
				n, err := rxp.readBuffer.Write(read.Buf[:read.Size])
				if err != nil {
					return 0, errors.Wrap(err, "buffer")
				}
				if n != read.Size {
					return 0, errors.New("short buffer")
				}
			} else {
				close(rxp.reads)
				return 0, io.EOF
			}

		default:
			break preread
		}
	}
	if rxp.readBuffer.Len() > 0 {
		return rxp.readBuffer.Read(p)

	} else {
		read, ok := <-rxp.reads
		if !ok {
			return 0, io.EOF
		}
		if !read.Eof {
			n, err := rxp.readBuffer.Write(read.Buf[:read.Size])
			if err != nil {
				return 0, errors.Wrap(err, "buffer")
			}
			if n != read.Size {
				return 0, errors.Wrap(err, "short buffer")
			}
			rxp.readPool.Put(read.Buf)

			return rxp.readBuffer.Read(p)

		} else {
			close(rxp.reads)
			return 0, io.EOF
		}
	}
}

func (rxp *RxPortal) Rx(wm *WireMessage) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.Wrap(err, "send on closed rxs")
		}
	}()
	rxp.rxs <- wm
	return err
}

func (rxp *RxPortal) Close() {
	if !rxp.closed {
		rxp.reads <- &RxRead{nil, 0, true}
		rxp.closed = true
		close(rxp.rxs)
	}
}

func (rxp *RxPortal) run() {
}
