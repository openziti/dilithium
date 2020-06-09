package westworld2

import (
	"github.com/michaelquigley/dilithium/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"net"
	"time"
)

type listenerConn struct {
	conn     *net.UDPConn
	peer     *net.UDPAddr
	rxQueue  chan *wireMessage
	seq      *util.Sequence
	txPortal2 *txPortal2
	rxPortal2 *rxPortal2
	pool     *pool
	ins      Instrument
}

func newListenerConn(conn *net.UDPConn, peer *net.UDPAddr, ins Instrument) *listenerConn {
	lc := &listenerConn{
		conn:    conn,
		peer:    peer,
		rxQueue: make(chan *wireMessage, rxQueueSize),
		seq:     util.NewSequence(0),
		pool:    newPool("listenerConn", ins),
		ins:     ins,
	}
	lc.txPortal2 = newTxPortal2(conn, peer, ins)
	lc.rxPortal2 = newRxPortal2(conn, peer, ins)
	go lc.rxer()
	return lc
}

func (self *listenerConn) Read(p []byte) (int, error) {
	return self.rxPortal2.read(p)
}

func (self *listenerConn) Write(p []byte) (int, error) {
	self.txPortal2.queueTx(newData(self.seq.Next(), p, self.pool))
	if err := self.txPortal2.txError(); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (self *listenerConn) Close() error {
	return nil
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

		if wm.mt == DATA {
			if wm.ack != -1 {
				self.txPortal2.ack(wm.ack)
			}
			self.rxPortal2.rx(wm)

		} else if wm.mt == ACK {
			if wm.ack != -1 {
				self.txPortal2.ack(wm.ack)
			}
			wm.buffer.unref()

		} else {
			wm.buffer.unref()

			if self.ins != nil {
				self.ins.unexpectedMessageType(self.peer, wm.mt)
			}
		}
	}
}

func (self *listenerConn) hello(hello *wireMessage) error {
	/*
	 * Receive Hello
	 */
	self.rxPortal2.setAccepted(hello.seq)
	hello.buffer.unref()
	/* */

	/*
	 * Send Ack'd Hello
	 */
	helloAckSeq := self.seq.Next()
	helloAck := newHelloAck(helloAckSeq, hello.seq, self.pool)
	defer helloAck.buffer.unref()

	if err := writeWireMessage(helloAck, self.conn, self.peer, self.ins); err != nil {
		return errors.Wrap(err, "write hello ack")
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

		if ack.mt == ACK && ack.ack == helloAckSeq {
			return nil
		}
		return errors.New("invalid hello ack")

	case <-time.After(5 * time.Second):
		return errors.New("timeout")
	}
	/* */
}
