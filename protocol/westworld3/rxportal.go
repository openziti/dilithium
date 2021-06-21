package westworld3

import (
	"bytes"
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
	closer     *closer
	profile    *Profile
	closed     bool
	ii         InstrumentInstance
}

type rxRead struct {
	buf []byte
	sz  int
	eof bool
}

func newRxPortal(conn *net.UDPConn, peer *net.UDPAddr, txPortal *txPortal, seq *util.Sequence, closer *closer, profile *Profile, ii InstrumentInstance) *rxPortal {
	rx := &rxPortal{
		tree:       btree.NewWith(profile.RxPortalTreeLen, utils.Int32Comparator),
		accepted:   -1,
		rxs:        make(chan *wireMessage),
		reads:      make(chan *rxRead, profile.ReadsQueueLen),
		readBuffer: new(bytes.Buffer),
		readPool:   new(sync.Pool),
		ackPool:    newPool("ackPool", uint32(profile.PoolBufferSz), ii),
		conn:       conn,
		peer:       peer,
		txPortal:   txPortal,
		seq:        seq,
		closer:     closer,
		profile:    profile,
		ii:         ii,
	}
	rx.readPool.New = func() interface{} {
		return make([]byte, profile.PoolBufferSz)
	}
	go rx.run()
	return rx
}

func (self *rxPortal) read(p []byte) (int, error) {
preread:
	for {
		select {
		case read, ok := <-self.reads:
			if !ok {
				logrus.Error("!ok(1)")
				return 0, io.EOF
			}
			if !read.eof {
				n, err := self.readBuffer.Write(read.buf[:read.sz])
				if err != nil {
					return 0, errors.Wrap(err, "buffer")
				}
				if n != read.sz {

				}
			} else {
				logrus.Error("read EOF(1)")
				return 0, io.EOF
			}

		default:
			break preread
		}
	}
	if self.readBuffer.Len() > 0 {
		return self.readBuffer.Read(p)
	} else {
		read, ok := <-self.reads
		if !ok {
			logrus.Error("!ok")
			return 0, io.EOF
		}
		if !read.eof {
			n, err := self.readBuffer.Write(read.buf[:read.sz])
			if err != nil {
				return 0, errors.Wrap(err, "buffer")
			}
			if n != read.sz {
				return 0, errors.Wrap(err, "short buffer")
			}
			self.readPool.Put(read.buf)

			return self.readBuffer.Read(p)

		} else {
			logrus.Errorf("close notified")
			close(self.reads)
			return 0, io.EOF
		}
	}
}

func (self *rxPortal) rx(wm *wireMessage) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.Wrap(err, "send on closed rxs")
		}
	}()
	self.rxs <- wm
	return err
}

func (self *rxPortal) setAccepted(accepted int32) {
	self.accepted = accepted
}

func (self *rxPortal) close() {
	if !self.closed {
		self.reads <- &rxRead{nil, 0, true}
		self.closed = true
		close(self.rxs)
	}
}

func (self *rxPortal) run() {
	logrus.Infof("started")
	defer logrus.Warn("exited")

	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("recovered (%v)", r)
		}
	}()

	for {
		var wm *wireMessage
		var ok bool
		select {
		case wm, ok = <-self.rxs:
			if !ok {
				return
			}

		case <-time.After(time.Duration(self.profile.ConnectionInactiveTimeoutMs) * time.Millisecond):
			self.closer.timeout()
			return
		}

		switch wm.messageType() {
		case DATA:
			_, found := self.tree.Get(wm.seq)
			if !found && (wm.seq > self.accepted || (wm.seq == 0 && self.accepted == math.MaxInt32)) {
				if sz, err := wm.asDataSize(); err == nil {
					self.tree.Put(wm.seq, wm)
					self.rxPortalSz += int(sz)
					self.ii.RxPortalSzChanged(self.peer, self.rxPortalSz)
				} else {
					logrus.Errorf("unexpected mt [%d] (%v)", wm.messageType(), err)
				}
			} else {
				self.ii.DuplicateRx(self.peer, wm)
			}

			var rtt *uint16
			if wm.hasFlag(RTT) {
				if _, rttIn, err := wm.asData(); err == nil {
					rtt = rttIn
				} else {
					logrus.Errorf("unexpected mt [%d] (%v)", wm.messageType(), err)
				}
			}

			if ack, err := newAck([]ack{{wm.seq, wm.seq}}, int32(self.rxPortalSz), rtt, self.ackPool); err == nil {
				if err := writeWireMessage(ack, self.conn, self.peer); err != nil {
					logrus.Errorf("error sending ack (%v)", err)
				}
				self.ii.WireMessageTx(self.peer, ack)
				self.ii.TxAck(self.peer, ack)
				ack.buffer.unref()
			}

			if found {
				wm.buffer.unref()
			}

			if self.tree.Size() > 0 {
				startingRxPortalSz := self.rxPortalSz

				var next int32
				if self.accepted < math.MaxInt32 {
					next = self.accepted + 1
				} else {
					next = 0
				}

				keys := self.tree.Keys()
				for _, key := range keys {
					if key.(int32) == next {
						v, _ := self.tree.Get(key)
						wm := v.(*wireMessage)
						buf := self.readPool.Get().([]byte)
						if data, _, err := wm.asData(); err == nil {
							n := copy(buf, data)
							self.reads <- &rxRead{buf, n, false}

							self.tree.Remove(key)
							self.rxPortalSz -= len(data)
							self.ii.RxPortalSzChanged(self.peer, self.rxPortalSz)
							wm.buffer.unref()
							self.accepted = next
							if next < math.MaxInt32 {
								next++
							} else {
								next = 0
							}
						} else {
							logrus.Errorf("unexpected mt [%d]", wm.mt)
						}
					}
				}

				/*
				 * Send "pacing" KEEPALIVE when buffer size changes more than RxPortalSzPacingThresh.
				 */
				if startingRxPortalSz > self.profile.TxPortalMinSz && float64(self.rxPortalSz)/float64(startingRxPortalSz) < self.profile.RxPortalSzPacingThresh {
					if keepalive, err := newKeepalive(self.rxPortalSz, self.ackPool); err == nil {
						if err := writeWireMessage(keepalive, self.conn, self.peer); err != nil {
							logrus.Errorf("error sending pacing keepalive (%v)", err)
						}
						self.ii.WireMessageTx(self.peer, keepalive)
						self.ii.TxKeepalive(self.peer, keepalive)
						keepalive.buffer.unref()
					}
				}
			}

		case KEEPALIVE:
			//

		case CLOSE:
			closeAck, err := newAck([]ack{{wm.seq, wm.seq}}, int32(self.rxPortalSz), nil, self.ackPool)
			if err == nil {
				if err := writeWireMessage(closeAck, self.conn, self.peer); err != nil {
					logrus.Errorf("error writing close ack (%v)", err)
				}
				self.ii.WireMessageTx(self.peer, closeAck)
				self.ii.TxAck(self.peer, closeAck)
			} else {
				logrus.Errorf("error creating close ack (%v)", err)
			}
			self.closer.rxCloseSeqIn <- wm.seq
			wm.buffer.unref()

		default:
			logrus.Errorf("unexpected message type [%d]", wm.messageType())
			wm.buffer.unref()
		}
	}
}
