package dilithium

import (
	"bytes"
	"github.com/emirpasic/gods/trees/btree"
	"github.com/emirpasic/gods/utils"
	"github.com/openziti/dilithium/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"io"
	"math"
	"sync"
	"time"
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

func (rxp *RxPortal) SetAccepted(accepted int32) {
	rxp.accepted = accepted
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
	logrus.Info("started")
	defer logrus.Warn("exited")

	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("recovered (%v)", r)
		}
	}()

	for {
		var wm *WireMessage
		var ok bool
		select {
		case wm, ok = <-rxp.rxs:
			if !ok {
				return
			}

		case <-time.After(time.Duration(rxp.txp.alg.Profile().ConnectionTimeout) * time.Millisecond):
			// rxp.closer.timeout()
			return
		}

		switch wm.messageType() {
		case DATA:
			_, found := rxp.tree.Get(wm.Seq)
			if !found && (wm.Seq > rxp.accepted || (wm.Seq == 0 && rxp.accepted == math.MaxInt32)) {
				if size, err := wm.asDataSize(); err == nil {
					rxp.tree.Put(wm.Seq, wm)
					rxp.rxPortalSize += int(size)
				} else {
					logrus.Errorf("unexpected as data size (%v)", err)
				}
			}

			var rtt *uint16
			if wm.hasFlag(RTT) {
				if _, rttIn, err := wm.asData(); err == nil {
					rtt = rttIn
				} else {
					logrus.Errorf("unexpected as data (%v)", err)
				}
			}

			if ack, err := newAck([]Ack{{wm.Seq, wm.Seq}}, int32(rxp.rxPortalSize), rtt, rxp.ackPool); err == nil {
				if err := writeWireMessage(ack, rxp.transport); err != nil {
					logrus.Errorf("error sending ack (%v)", err)
				}
				ack.buf.Unref()
			}

			if found {
				wm.buf.Unref()
			}

			if rxp.tree.Size() > 0 {
				startingRxPortalSize := rxp.rxPortalSize

				var next int32
				if rxp.accepted < math.MaxInt32 {
					next = rxp.accepted + 1
				} else {
					next = 0
				}

				keys := rxp.tree.Keys()
				for _, key := range keys {
					if key.(int32) == next {
						v, _ := rxp.tree.Get(key)
						wm := v.(*WireMessage)
						buf := rxp.readPool.Get().([]byte)
						if data, _, err := wm.asData(); err == nil {
							n := copy(buf, data)
							rxp.reads <- &RxRead{buf, n, false}

							rxp.tree.Remove(key)
							rxp.rxPortalSize -= len(data)
							wm.buf.Unref()
							rxp.accepted = next
							if next < math.MaxInt32 {
								next++
							} else {
								next = 0
							}
						} else {
							logrus.Errorf("unexpected mt [%d]", wm.Mt)
						}
					}
				}

				// Send "pacing" KEEPALIVE?
				if rxp.txp.alg.RxPortalPacing(startingRxPortalSize, rxp.rxPortalSize) {
					if keepalive, err := newKeepalive(rxp.rxPortalSize, rxp.ackPool); err == nil {
						if err := writeWireMessage(keepalive, rxp.transport); err != nil {
							logrus.Errorf("error sending pacing keepalive (%v)", err)
						}
						keepalive.buf.Unref()
					}
				}
			}

		case KEEPALIVE:
			wm.buf.Unref()

		case CLOSE:
			if closeAck, err := newAck([]Ack{{wm.Seq, wm.Seq}}, int32(rxp.rxPortalSize), nil, rxp.ackPool); err == nil {
				if err := writeWireMessage(closeAck, rxp.transport); err != nil {
					logrus.Errorf("error writing close ack (%v)", err)
				}
			} else {
				logrus.Errorf("error creating close ack (%v)", err)
			}
			wm.buf.Unref()

		default:
			logrus.Errorf("unexpected message type [%d]", wm.messageType())
			wm.buf.Unref()
		}
	}
}
