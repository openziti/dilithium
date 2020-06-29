package westworld2

import (
	"github.com/michaelquigley/dilithium/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
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
}

func newDialerConn(conn *net.UDPConn, peer *net.UDPAddr, config *Config) *dialerConn {
	dc := &dialerConn{
		conn:   conn,
		peer:   peer,
		seq:    util.NewSequence(0),
		pool:   newPool("dialerConn", config),
		config: config,
	}
	dc.txPortal = newTxPortal(conn, peer, config)
	dc.rxPortal = newRxPortal(conn, peer, dc.txPortal, dc.seq, config)
	return dc
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
		wm, _, err := readWireMessage(self.conn, self.pool, self.config.i)
		if err != nil {
			if self.config.i != nil {
				self.config.i.readError(self.peer, err)
			}
			continue
		}

		if wm.mt == DATA || wm.mt == CLOSE {
			if wm.ack != -1 {
				self.txPortal.ack(wm.ack)
			}
			self.rxPortal.rx(wm)

		} else if wm.mt == ACK {
			if wm.ack != -1 {
				self.txPortal.ack(wm.ack)
			}
			if wm.mf & RTT == 1 {
				ts, err := wm.readRtt()
				if err == nil {
					self.txPortal.rtt(ts)
				} else {
					logrus.Errorf("error reading rtt (%v)", err)
				}
			}
			wm.buffer.unref()

		} else {
			if self.config.i != nil {
				self.config.i.unexpectedMessageType(self.peer, wm.mt)
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

	if err := writeWireMessage(hello, self.conn, self.peer, self.config.i); err != nil {
		return errors.Wrap(err, "write hello")
	}
	/* */

	/*
	 * Expect Ack'd Hello Response
	 */
	if err := self.conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return errors.Wrap(err, "set read deadline")
	}

	helloAck, _, err := readWireMessage(self.conn, self.pool, self.config.i)
	if err != nil {
		return errors.Wrap(err, "read hello ack")
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
	ack := newAck(helloAck.seq, self.pool)
	defer ack.buffer.unref()

	if err := writeWireMessage(ack, self.conn, self.peer, self.config.i); err != nil {
		return errors.Wrap(err, "write ack")
	}
	/* */

	if self.config.i != nil {
		self.config.i.connected(self.peer)
	}

	go self.rxer()

	return nil
}
