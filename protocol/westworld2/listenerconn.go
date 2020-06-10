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
	txPortal *txPortal
	rxPortal *rxPortal
	pool     *pool
	ins      Instrument
}

func newListenerConn(conn *net.UDPConn, peer *net.UDPAddr, ins Instrument) *listenerConn {
	lc := &listenerConn{
		conn:    conn,
		peer:    peer,
		rxQueue: make(chan *wireMessage, listenerRxQueueSz),
		seq:     util.NewSequence(0),
		pool:    newPool("listenerConn", ins),
		ins:     ins,
	}
	lc.txPortal = newTxPortal(conn, peer, ins)
	lc.rxPortal = newRxPortal(conn, peer, ins)
	return lc
}

func (self *listenerConn) Read(p []byte) (int, error) {
	return self.rxPortal.read(p)
}

func (self *listenerConn) Write(p []byte) (int, error) {
	if err := self.txPortal.tx(newData(self.seq.Next(), p, self.pool)); err != nil {
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
				self.txPortal.ack(wm.ack)
			}
			self.rxPortal.rx(wm)

		} else if wm.mt == ACK {
			if wm.ack != -1 {
				self.txPortal.ack(wm.ack)
			}
			wm.buffer.unref()

		} else {
			if self.ins != nil {
				self.ins.unexpectedMessageType(self.peer, wm.mt)
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
