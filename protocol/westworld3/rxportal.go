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
	ii         InstrumentInstance
}

type rxRead struct {
	buf []byte
	sz  int
	eof bool
}

func newRxPortal(conn *net.UDPConn, peer *net.UDPAddr, txPortal *txPortal, seq *util.Sequence, profile *Profile, ii InstrumentInstance) *rxPortal {
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
				return 0, io.EOF
			}
			if !read.eof {
				n, err := self.readBuffer.Write(read.buf[:read.sz])
				if err != nil {
					return 0, errors.Wrap(err, "buffer")
				}
				if n != read.sz {

				}
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
			logrus.Infof("close notified")
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

func (self *rxPortal) run() {
	logrus.Infof("started")
	defer logrus.Warn("exited")

	/*
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("recovered (%v)", r)
		}
	}()
	*/

	for {
		wm, ok := <-self.rxs
		if !ok {
			return
		}

		if wm.mt == DATA {
			_, found := self.tree.Get(wm.seq)
			if !found && (wm.seq > self.accepted || (wm.seq == 0 && self.accepted == math.MaxInt32)) {
				if sz, err := wm.asDataSize(); err == nil {
					self.tree.Put(wm.seq, wm)
					self.rxPortalSz += int(sz)
				} else {
					logrus.Errorf("unexpected mt [%d]", wm.mt)
				}
			} else {
				self.ii.DuplicateRx(self.peer, wm)
				wm.buffer.unref()
			}

			var rtt *uint16
			if wm.hasFlag(RTT) {
				if _, rttIn, err := wm.asData(); err == nil {
					rtt = rttIn
				} else {
					logrus.Errorf("unexpected mt [%d]", wm.mt)
				}
			}

			if ack, err := newAck([]ack{{wm.seq, wm.seq}}, int32(self.rxPortalSz), rtt, self.ackPool); err == nil {
				if err := writeWireMessage(ack, self.conn, self.peer); err != nil {
					logrus.Errorf("error sending ack (%v)", err)
				}
				self.ii.WireMessageTx(self.peer, ack)
				ack.buffer.unref()
			}

			if self.tree.Size() > 0 {
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
			}
		} else if wm.mt == CLOSE && !self.closed {
			self.txPortal.close(self.seq)
			self.reads <- &rxRead{nil, 0, true}
			self.closed = true
			close(self.rxs)
		}
	}
}
