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
}

func newListenerConn(conn *net.UDPConn, peer *net.UDPAddr, ins instrument) *listenerConn {
	lc := &listenerConn{
		conn:    conn,
		peer:    peer,
		rxQueue: make(chan *wireMessage, rxQueueSize),
		seq:     util.NewSequence(0),
		pool:    newPool("listenerConn"),
	}
	lc.txPortal = newTxPortal(conn, peer, ins)
	lc.rxPortal = newRxPortal(lc.txPortal.txAcks)
	go lc.rxer()
	return lc
}

func (self *listenerConn) Read(p []byte) (int, error) {
	rxdr, ok := <-self.rxPortal.rxDataQueue
	if !ok {
		return 0, errors.New("closed")
	}
	n := copy(p, rxdr.buf[:rxdr.sz])
	self.rxPortal.rxDataPool.Put(rxdr.buf)
	return n, nil
}

func (self *listenerConn) Write(p []byte) (int, error) {
	self.txPortal.txQueue <- newData(self.seq.Next(), p, self.pool)

	select {
	case err := <-self.txPortal.txErrors:
		return 0, err
	default:
		// no error
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
				self.txPortal.rxAcks <- wm.ack
			}
			self.rxPortal.rxWmQueue <- wm

		} else if wm.mt == ACK {
			if wm.ack != -1 {
				self.txPortal.rxAcks <- wm.ack
			}
			wm.buffer.unref()

		} else {
			logrus.Errorf("invalid mt [%s]", mtString(wm.mt))
			wm.buffer.unref()
		}
	}
}

func (self *listenerConn) hello(hello *wireMessage) error {
	logrus.Infof("{hello} <- [%s]", self.peer)

	self.rxPortal.accepted = hello.seq
	hello.buffer.unref()

	helloAckSeq := self.seq.Next()
	helloAck := newHelloAck(helloAckSeq, hello.seq, self.pool)
	defer helloAck.buffer.unref()

	if err := writeWireMessage(helloAck, self.conn, self.peer); err != nil {
		return errors.Wrap(err, "write hello ack")
	}
	logrus.Infof("{helloack} -> [%s]", self.peer)

	select {
	case ack, ok := <-self.rxQueue:
		if !ok {
			return errors.New("rxQueue closed")
		}
		defer ack.buffer.unref()

		if ack.mt == ACK && ack.ack == helloAckSeq {
			logrus.Infof("{ack} <- [%s]", self.peer)
			logrus.Infof("connection established, peer [%s]", self.peer)
			return nil
		}
		return errors.New("invalid hello ack")

	case <-time.After(5 * time.Second):
		return errors.New("timeout")
	}
}
