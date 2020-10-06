package westworld2

import (
	"crypto/rand"
	"github.com/michaelquigley/dilithium/util"
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
	config   *Config
	ii       InstrumentInstance
}

func newListenerConn(listener *listener, conn *net.UDPConn, peer *net.UDPAddr, config *Config) (*listenerConn, error) {
	sSeq := int64(0)
	if config.seqRandom {
		randSeq, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt32))
		if err != nil {
			return nil, errors.Wrap(err, "random sequence")
		}
		sSeq = randSeq.Int64()
	}
	lc := &listenerConn{
		listener: listener,
		conn:     conn,
		peer:     peer,
		rxQueue:  make(chan *wireMessage, config.listenerRxQLen),
		seq:      util.NewSequence(int32(sSeq)),
		config:   config,
	}
	if config.i != nil {
		lc.ii = config.i.newInstance(peer)
		lc.ii.listener(listener.addr)
	}
	lc.pool = newPool("listenerConn", lc.ii)
	lc.txPortal = newTxPortal(conn, peer, config, lc.ii)
	lc.rxPortal = newRxPortal(conn, peer, lc.txPortal, lc.seq, config, lc.ii)
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
	logrus.Info("started")
	defer logrus.Warn("exited")

	for {
		wm, ok := <-self.rxQueue
		if !ok {
			return
		}

		if wm.mt == DATA || wm.mt == CLOSE {
			if wm.ack != -1 {
				self.txPortal.ack(self.txPortal.peer, wm.ack, -1)
			}
			self.rxPortal.rx(wm)

		} else if wm.mt == ACK {
			if wm.ack != -1 {
				rxPortalSz := -1
				if len(wm.data) == 4 {
					rxPortalSz = int(util.ReadInt32(wm.data[0:4]))
				}
				self.txPortal.ack(self.txPortal.peer, wm.ack, int(rxPortalSz))
			}
			if wm.mf&RTT == 1 {
				ts, err := wm.readRtt()
				if err == nil {
					self.txPortal.rtt(ts)
				} else {
					logrus.Errorf("error reading rtt (%v)", err)
				}
			}
			wm.buffer.unref()

		} else {
			if self.ii != nil {
				self.ii.unexpectedMessageType(self.peer, wm.mt)
			}
			wm.buffer.unref()
		}
	}
}

func (self *listenerConn) hello(hello *wireMessage) error {
	/*
	 * Receive Hello
	 */
	self.rxPortal.setAccepted(hello.seq)
	hello.buffer.unref()
	/* */

	/*
	 * Send Ack'd Hello
	 */
	helloAckSeq := self.seq.Next()
	helloAck := newHelloAck(helloAckSeq, hello.seq, self.pool)
	defer helloAck.buffer.unref()

	if err := writeWireMessage(helloAck, self.conn, self.peer); err != nil {
		return errors.Wrap(err, "write hello ack")
	}
	if self.ii != nil {
		self.ii.wireMessageTx(self.peer, helloAck)
	}
	/* */

	/*
	 * Receive Final Ack
	 */
	select {
	case ack, ok := <-self.rxQueue:
		if !ok {
			return errors.New("rxQueue closed")
		}
		defer ack.buffer.unref()

		if ack.mt != ACK {
			return errors.Errorf("expected ack, got (%d)", ack.mt)
		}
		if ack.ack != helloAckSeq {
			return errors.Errorf("invalid hello ack sequence (%d != %d)", ack.ack, helloAckSeq)
		}
		return nil

	case <-time.After(5 * time.Second):
		return errors.New("timeout")
	}
	/* */
}
