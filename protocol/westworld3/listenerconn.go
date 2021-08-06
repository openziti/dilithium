package westworld3

import (
	"crypto/rand"
	"fmt"
	"github.com/openziti/dilithium/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"math"
	"math/big"
	"net"
	"sync/atomic"
	"time"
)

type listenerConn struct {
	listener      *listener
	conn          *net.UDPConn
	peer          *net.UDPAddr
	rxQueue       chan *wireMessage
	rxQueueClosed int32
	seq           *util.Sequence
	txPortal      *txPortal
	rxPortal      *rxPortal
	closer        *closer
	pool          *pool
	profile       *Profile
	ii            InstrumentInstance
}

func newListenerConn(listener *listener, conn *net.UDPConn, peer *net.UDPAddr, profile *Profile, callerHook func()) (*listenerConn, error) {
	startSeq := int64(0)
	if profile.RandomizeSeq {
		randomSeq, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt32))
		if err != nil {
			return nil, errors.Wrap(err, "randomize sequence")
		}
		startSeq = randomSeq.Int64()
	}
	lc := &listenerConn{
		listener:      listener,
		conn:          conn,
		peer:          peer,
		rxQueue:       make(chan *wireMessage, profile.ListenerRxQueueLen),
		rxQueueClosed: 0,
		seq:           util.NewSequence(int32(startSeq)),
		profile:       profile,
	}
	id := fmt.Sprintf("listenerConn_%s_%s", listener.addr, peer)
	lc.ii = profile.i.NewInstance(id, peer)
	lc.pool = newPool(id, uint32(dataStart+profile.MaxSegmentSz), lc.ii)
	closeHook := func() {
		lc.ii.Shutdown()
		if atomic.CompareAndSwapInt32(&lc.rxQueueClosed, 0, 1) {
			close(lc.rxQueue)
		}
		if callerHook != nil {
			callerHook()
		}
	}
	lc.closer = newCloser(lc.seq, lc.profile, closeHook)
	lc.txPortal = newTxPortal(conn, peer, lc.closer, profile, lc.pool, lc.ii)
	lc.rxPortal = newRxPortal(conn, peer, lc.txPortal, lc.seq, lc.closer, profile, lc.ii)
	lc.closer.txPortal = lc.txPortal
	lc.closer.rxPortal = lc.rxPortal
	return lc, nil
}

func (self *listenerConn) Read(p []byte) (int, error) {
	return self.rxPortal.read(p)
}

func (self *listenerConn) Write(p []byte) (int, error) {
	return self.txPortal.tx(p, self.seq)
}

func (self *listenerConn) Close() error {
	logrus.Warnf("close requested")
	return self.txPortal.sendClose(self.seq)
}

func (self *listenerConn) RemoteAddr() net.Addr {
	return self.peer
}

func (self *listenerConn) LocalAddr() net.Addr {
	return self.conn.LocalAddr()
}

func (self *listenerConn) SetDeadline(t time.Time) error {
	return nil
}

func (self *listenerConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (self *listenerConn) SetWriteDeadline(t time.Time) error {
	return nil
}

func (self *listenerConn) queue(wm *wireMessage) {
	self.rxQueue <- wm
}

func (self *listenerConn) rxer() {
	logrus.Infof("started")
	defer logrus.Warn("exited")

	for {
		wm, ok := <-self.rxQueue
		if !ok {
			return
		}
		self.ii.WireMessageRx(self.peer, wm)

		switch wm.messageType() {
		case DATA:
			_, rttTs, err := wm.asData()
			if err != nil {
				logrus.Errorf("as data error (%v)", err)
				continue
			}
			if rttTs != nil {
				self.txPortal.rtt(*rttTs)
			}
			if err := self.rxPortal.rx(wm); err != nil {
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
				self.txPortal.rtt(*rttTs)
			}
			self.txPortal.updateRxPortalSz(int(rxPortalSz))
			if err := self.txPortal.ack(acks); err != nil {
				logrus.Errorf("error acking (%v)", err)
				continue
			}
			self.ii.RxAck(self.peer, wm)
			wm.buffer.unref()

		case KEEPALIVE:
			rxPortalSz, err := wm.asKeepalive()
			if err != nil {
				logrus.Errorf("as keepalive error (%v)", err)
				continue
			}
			self.txPortal.updateRxPortalSz(rxPortalSz)
			if err := self.rxPortal.rx(wm); err != nil {
				logrus.Errorf("error forwarding keepalive to rxPortal (%v)", err)
				continue
			}
			self.ii.RxKeepalive(self.peer, wm)
			wm.buffer.unref()

		case CLOSE:
			if err := self.rxPortal.rx(wm); err != nil {
				logrus.Errorf("error rx-ing close (%v)", err)
			}

		default:
			logrus.Errorf("unexpected message type: %d", wm.mt)
			self.ii.UnexpectedMessageType(self.peer, wm.mt)
			wm.buffer.unref()
		}
	}
}

func (self *listenerConn) hello(wm *wireMessage) error {
	logrus.Infof("starting hello process")
	defer logrus.Infof("completed hello process")

	// Receive Hello
	if hello, _, err := wm.asHello(); err == nil {
		self.rxPortal.setAccepted(wm.seq)
		wm.buffer.unref()

		helloAckSeq := self.seq.Next()
		helloAck, err := newHello(helloAckSeq, hello, &Ack{wm.seq, wm.seq}, self.pool)
		if err != nil {
			err = errors.Wrap(err, "new hello")
			self.ii.ConnectionError(self.peer, err)
			return err
		}
		defer helloAck.buffer.unref()

		for i := 0; i < 5; i++ {
			// Send Hello Ack
			if err := writeWireMessage(helloAck, self.conn, self.peer); err != nil {
				err = errors.Wrap(err, "write hello ack")
				self.ii.ConnectionError(self.peer, err)
				return err
			}
			self.ii.WireMessageTx(self.peer, helloAck)

			// Receive Response Ack
			select {
			case ackWm, ok := <-self.rxQueue:
				if !ok {
					err = errors.New("rx queue closed")
					self.ii.ConnectionError(self.peer, err)
					return err
				}
				defer ackWm.buffer.unref()
				self.ii.WireMessageRx(self.peer, ackWm)

				if ackWm.mt != ACK {
					logrus.Errorf("expected ACK, got [%d]", ackWm.messageType())
					continue
				}
				if ack, _, _, err := ackWm.asAck(); err == nil {
					if len(ack) == 1 {
						if ack[0].Start != helloAckSeq {
							logrus.Errorf("invalid hello ack sequence (%d != %d)", ack[0].Start, helloAckSeq)
							continue
						}

						// connection established, now we can start
						go self.rxer()
						go self.txPortal.start()
						go self.closer.run()

						return nil
					}
				}

			case <-time.After(5 * time.Second):
				logrus.Infof("timeout")
			}
		}

		err = errors.New("connection failed")
		self.ii.ConnectionError(self.peer, err)
		return err

	} else {
		err = errors.Wrap(err, "expected hello")
		self.ii.ConnectionError(self.peer, err)
		return err
	}
}
