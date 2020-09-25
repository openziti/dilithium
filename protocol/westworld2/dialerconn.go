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

type dialerConn struct {
	conn     *net.UDPConn
	peer     *net.UDPAddr
	seq      *util.Sequence
	txPortal *txPortal
	rxPortal *rxPortal
	pool     *pool
	config   *Config
	ii       InstrumentInstance
}

func newDialerConn(conn *net.UDPConn, peer *net.UDPAddr, config *Config) (*dialerConn, error) {
	sSeq := int64(0)
	if config.seqRandom {
		randSeq, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt32))
		if err != nil {
			return nil, errors.Wrap(err, "random sequence")
		}
		sSeq = randSeq.Int64()
	}
	dc := &dialerConn{
		conn:   conn,
		peer:   peer,
		seq:    util.NewSequence(int32(sSeq)),
		config: config,
	}
	if config.i != nil {
		dc.ii = config.i.newInstance(peer)
	}
	dc.pool = newPool("dialerConn", dc.ii)
	dc.txPortal = newTxPortal(conn, peer, config, dc.ii)
	dc.rxPortal = newRxPortal(conn, peer, dc.txPortal, dc.seq, config, dc.ii)
	return dc, nil
}

func (self *dialerConn) Read(p []byte) (int, error) {
	return self.rxPortal.read(p)
}

func (self *dialerConn) Write(p []byte) (int, error) {
	return self.txPortal.tx(p, self.seq)
}

func (self *dialerConn) Close() error {
	logrus.Infof("close requested")
	return self.txPortal.close(self.seq)
}

func (self *dialerConn) RemoteAddr() net.Addr {
	return self.peer
}

func (self *dialerConn) LocalAddr() net.Addr {
	return self.conn.LocalAddr()
}

func (self *dialerConn) SetDeadline(t time.Time) error {
	return nil
}

func (self *dialerConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (self *dialerConn) SetWriteDeadline(t time.Time) error {
	return nil
}

func (self *dialerConn) rxer() {
	logrus.Info("started")
	defer logrus.Warn("exited")

	for {
		wm, peer, err := readWireMessage(self.conn, self.pool)
		if err != nil {
			if self.ii != nil {
				self.ii.readError(self.peer, err)
			}
			continue
		}
		if self.ii != nil {
			self.ii.wireMessageRx(peer, wm)
		}

		if wm.mt == DATA || wm.mt == CLOSE {
			if wm.ack != -1 {
				self.txPortal.ack(peer, wm.ack, -1)
			}
			if err := self.rxPortal.rx(wm); err != nil {
				logrus.Errorf("error writing (%v)", err)
				return
			}

		} else if wm.mt == ACK {
			if wm.ack != -1 {
				rxPortalSz := int32(-1)
				if len(wm.data) == 4 {
					rxPortalSz = util.ReadInt32(wm.data[0:4])
				}
				self.txPortal.ack(peer, wm.ack, int(rxPortalSz))
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

func (self *dialerConn) hello() error {
	/*
	 * Send Hello
	 */
	helloSeq := self.seq.Next()
	hello := newHello(helloSeq, self.pool)
	defer hello.buffer.unref()

	if err := writeWireMessage(hello, self.conn, self.peer); err != nil {
		return errors.Wrap(err, "write hello")
	}
	if self.ii != nil {
		self.ii.wireMessageTx(self.peer, hello)
	}
	/* */

	/*
	 * Expect Ack'd Hello Response
	 */
	if err := self.conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return errors.Wrap(err, "set read deadline")
	}

	helloAck, peer, err := readWireMessage(self.conn, self.pool)
	if err != nil {
		return errors.Wrap(err, "read hello ack")
	}
	if self.ii != nil {
		self.ii.wireMessageRx(peer, helloAck)
	}
	defer helloAck.buffer.unref()

	if helloAck.mt != HELLO {
		return errors.Wrap(err, "unexpected response")
	}
	if helloAck.ack != helloSeq {
		return errors.New("invalid hello ack")
	}
	if err := self.conn.SetReadDeadline(time.Time{}); err != nil {
		return errors.Wrap(err, "clear read deadline")
	}
	/* */

	// The next sequence should be the next highest sequence
	self.rxPortal.setAccepted(helloAck.seq)

	/*
	 * Send Final Ack
	 */
	ack := newAck(helloAck.seq, 0, self.pool)
	defer ack.buffer.unref()

	if err := writeWireMessage(ack, self.conn, self.peer); err != nil {
		return errors.Wrap(err, "write ack")
	}
	if self.ii != nil {
		self.ii.wireMessageTx(self.peer, ack)
	}
	/* */

	if self.ii != nil {
		self.ii.connected(self.peer)
	}

	go self.rxer()

	return nil
}
