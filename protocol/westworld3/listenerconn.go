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
	"time"
)

type listenerConn struct {
	listener *listener
	conn     *net.UDPConn
	peer     *net.UDPAddr
	rxQueue  chan *wireMessage
	seq      *util.Sequence
	txPortal *txPortal
	rxPortal *rxPortal
	pool     *pool
	profile  *Profile
	ii       InstrumentInstance
}

func newListenerConn(listener *listener, conn *net.UDPConn, peer *net.UDPAddr, profile *Profile) (*listenerConn, error) {
	startSeq := int64(0)
	if profile.RandomizeSeq {
		randomSeq, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt32))
		if err != nil {
			return nil, errors.Wrap(err, "randomize sequence")
		}
		startSeq = randomSeq.Int64()
	}
	lc := &listenerConn{
		listener: listener,
		conn:     conn,
		peer:     peer,
		rxQueue:  make(chan *wireMessage, profile.ListenerRxQueueLen),
		seq:      util.NewSequence(int32(startSeq)),
		profile:  profile,
	}
	id := fmt.Sprintf("listenerConn_%s", conn.LocalAddr())
	if profile.i != nil {
		lc.ii = profile.i.newInstance(id, peer)
	}
	lc.pool = newPool(id, uint32(dataStart+profile.MaxSegmentSz), lc.ii)
	lc.txPortal = newTxPortal(conn, peer, profile, lc.ii)
	lc.rxPortal = newRxPortal(conn, peer, lc.txPortal, lc.seq, profile)
	go lc.rxer()
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
	return self.txPortal.close(self.seq)
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

		if wm.mt == DATA || wm.mt == CLOSE {
			self.rxPortal.rx(wm)

		} else if wm.mt == ACK {
			if acks, rxPortalSz, rtt, err := wm.asAck(); err == nil {
				self.txPortal.updateRxPortalSz(int(rxPortalSz))
				if rtt != nil {
					self.txPortal.rtt(*rtt)
				}
				wm.buffer.unref()
				self.txPortal.ack(acks)

			} else {
				logrus.Errorf("error unmarshaling ack (%v)", err)
			}

		} else {
			// unexpected message type
			wm.buffer.unref()
		}
	}
}

func (self *listenerConn) hello(wm *wireMessage) error {
	// Receive Hello
	if hello, _, err := wm.asHello(); err == nil {
		self.rxPortal.setAccepted(wm.seq)
		wm.buffer.unref()

		helloAckSeq := self.seq.Next()
		helloAck, err := newHello(helloAckSeq, hello, &ack{wm.seq, wm.seq}, self.pool)
		if err != nil {
			return errors.Wrap(err, "new hello")
		}
		defer helloAck.buffer.unref()

		for i := 0; i < 5; i++ {
			// Send Hello Ack
			if err := writeWireMessage(helloAck, self.conn, self.peer); err != nil {
				return errors.Wrap(err, "write hello ack")
			}

			// Receive Response Ack
			select {
			case ackWm, ok := <-self.rxQueue:
				if !ok {
					return errors.New("rx queue closed")
				}
				defer ackWm.buffer.unref()

				if ackWm.mt != ACK {
					logrus.Errorf("expected ACK, got [%d]", ackWm.mt)
					continue
				}
				if ack, _, _, err := ackWm.asAck(); err == nil {
					if len(ack) == 1 {
						if ack[0].start != helloAckSeq {
							logrus.Errorf("invalid hello ack sequence (%d != %d)", ack[0].start, helloAckSeq)
							continue
						}
						return nil
					}
				}

			case <-time.After(5 * time.Second):
				logrus.Infof("timeout")
			}
		}

		return errors.New("connection failed")

	} else {
		return errors.Wrap(err, "expected hello")
	}
}
