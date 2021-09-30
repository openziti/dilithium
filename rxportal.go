package dilithium

import (
	"github.com/emirpasic/gods/trees/btree"
	"github.com/emirpasic/gods/utils"
	"github.com/openziti/dilithium/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"math"
	"time"
)

type RxPortal struct {
	adapter      Adapter
	sink         Sink
	tree         *btree.Tree
	accepted     int32
	rxs          chan *WireMessage
	rxPortalSize int
	readPool     *Pool
	ackPool      *Pool
	txp          *TxPortal
	seq          *util.Sequence
	closer       *Closer
	closed       bool
	ii           InstrumentInstance
}

type RxRead struct {
	Buf  []byte
	Size int
	Eof  bool
}

func NewRxPortal(adapter Adapter, sink Sink, txp *TxPortal, seq *util.Sequence, closer *Closer, ii InstrumentInstance) *RxPortal {
	rxp := &RxPortal{
		adapter:  adapter,
		sink:     sink,
		tree:     btree.NewWith(txp.alg.Profile().MaxTreeSize, utils.Int32Comparator),
		accepted: -1,
		rxs:      make(chan *WireMessage, 4),
		readPool: NewPool("readPool", uint32(txp.alg.Profile().PoolBufferSize), ii),
		ackPool:  NewPool("ackPool", uint32(txp.alg.Profile().PoolBufferSize), ii),
		txp:      txp,
		seq:      seq,
		closer:   closer,
		ii:       ii,
	}
	go rxp.run()
	go rxp.rxer()
	return rxp
}

func (rxp *RxPortal) SetAccepted(accepted int32) {
	rxp.accepted = accepted
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
		rxp.sink.Close()
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
					rxp.ii.RxPortalSzChanged(rxp.rxPortalSize)
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
					rxp.ii.WriteError(err)
				} else {
					rxp.ii.WireMessageTx(ack)
					rxp.ii.TxAck(ack)
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
						if data, _, err := wm.asData(); err == nil {
							if err := rxp.sink.Accept(data); err != nil {
								logrus.WithError(err).Error("write to data sink failed, exiting rx loop")
								return
							}

							rxp.tree.Remove(key)
							rxp.rxPortalSize -= len(data)
							rxp.ii.RxPortalSzChanged(rxp.rxPortalSize)
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
							rxp.ii.WriteError(err)
						} else {
							rxp.ii.WireMessageTx(keepalive)
							rxp.ii.TxKeepalive(keepalive)
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
					rxp.ii.WriteError(err)
				} else {
					rxp.ii.WireMessageTx(closeAck)
					rxp.ii.TxAck(closeAck)
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
			rxp.ii.ReadError(err)
			logrus.Errorf("error reading (%v)", err)
			rxp.closer.EmergencyStop()
			return
		}
		rxp.ii.WireMessageRx(wm)

		switch wm.messageType() {
		case DATA:
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
				now := time.Now().UnixNano()
				clockTs := uint16(now / int64(time.Millisecond))
				rttMs := clockTs - *rttTs
				rxp.txp.alg.UpdateRTT(int(rttMs))
			}
			rxp.txp.alg.UpdateRxPortalSize(int(rxPortalSz))
			if err := rxp.txp.ack(acks); err != nil {
				logrus.Errorf("error acking (%v)", err)
				continue
			}
			rxp.ii.RxAck(wm)
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
			rxp.ii.RxKeepalive(wm)
			wm.buf.Unref()

		case CLOSE:
			if err := rxp.Rx(wm); err != nil {
				logrus.Errorf("error rx-ing close (%v)", err)
			}

		default:
			logrus.Errorf("unexpected message type: %d", wm.messageType())
			wm.buf.Unref()
			rxp.ii.UnexpectedMessageType(wm.messageType())
		}
	}
}
