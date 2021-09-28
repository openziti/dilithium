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
)

type RxPortal struct {
	adapter      Adapter
	tree         *btree.Tree
	accepted     int32
	rxs          chan *WireMessage
	reads        chan *RxRead
	readBuffer   *bytes.Buffer
	rxPortalSize int
	rawReadPool  *sync.Pool
	readPool     *Pool
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

func NewRxPortal(adapter Adapter, txp *TxPortal, seq *util.Sequence, closer *Closer) *RxPortal {
	rxp := &RxPortal{
		adapter:     adapter,
		tree:        btree.NewWith(txp.alg.Profile().MaxTreeSize, utils.Int32Comparator),
		accepted:    -1,
		rxs:         make(chan *WireMessage),
		reads:       make(chan *RxRead, txp.alg.Profile().ReadsQueueSize),
		readBuffer:  new(bytes.Buffer),
		rawReadPool: new(sync.Pool),
		readPool:    NewPool("readPool", uint32(txp.alg.Profile().PoolBufferSize)),
		ackPool:     NewPool("ackPool", uint32(txp.alg.Profile().PoolBufferSize)),
		txp:         txp,
		seq:         seq,
		closer:      closer,
	}
	rxp.rawReadPool.New = func() interface{} {
		return make([]byte, txp.alg.Profile().PoolBufferSize)
	}
	go rxp.run()
	go rxp.rxer()
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
			rxp.rawReadPool.Put(read.Buf)

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
	select {
	case rxp.rxs <- wm:
	default:
		logrus.Info("dropped")
	}
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

			//case <-time.After(time.Duration(rxp.txp.alg.Profile().ConnectionTimeout) * time.Millisecond):
			// rxp.Closer.timeout()
			//return
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
				if err := writeWireMessage(ack, rxp.adapter); err != nil {
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
						buf := rxp.rawReadPool.Get().([]byte)
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
						if err := writeWireMessage(keepalive, rxp.adapter); err != nil {
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
				if err := writeWireMessage(closeAck, rxp.adapter); err != nil {
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

func (rxp *RxPortal) rxer() {
	logrus.Info("started")
	defer logrus.Warn("exited")

	for {
		wm, err := readWireMessage(rxp.adapter, rxp.readPool)
		if err != nil {
			logrus.Errorf("error reading (%v)", err)
			rxp.closer.EmergencyStop()
			return
		}

		switch wm.messageType() {
		case DATA:
			_, rttTs, err := wm.asData()
			if err != nil {
				logrus.Errorf("as data error (%v)", err)
				continue
			}
			if rttTs != nil {
				// self.txPortal.rtt(*rttTs)
			}
			if err := rxp.Rx(wm); err != nil {
				logrus.Errorf("error rx-ing (%v)", err)
				continue
			}

		case ACK:
			acks, rxPortalSz, rttTs, err := wm.asAck()
			if err != nil {
				logrus.Errorf("as ack error (%v)", err)
				continue
			}
			if rttTs != nil {
				//self.txPortal.rtt(*rttTs)
			}
			rxp.txp.alg.UpdateRxPortalSize(int(rxPortalSz))
			if err := rxp.txp.ack(acks); err != nil {
				logrus.Errorf("error acking (%v)", err)
				continue
			}
			wm.buf.Unref()

		case KEEPALIVE:
			rxPortalSz, err := wm.asKeepalive()
			if err != nil {
				logrus.Errorf("as keepalive error (%v)", err)
				continue
			}
			rxp.txp.alg.UpdateRxPortalSize(rxPortalSz)
			if err := rxp.Rx(wm); err != nil {
				logrus.Errorf("error forwarding keepalive to rxPortal (%v)", err)
				continue
			}
			wm.buf.Unref()

		case CLOSE:
			if err := rxp.Rx(wm); err != nil {
				logrus.Errorf("error rx-ing close (%v)", err)
			}

		default:
			logrus.Errorf("unexpected message type: %d", wm.messageType())
			wm.buf.Unref()
		}
	}
}
