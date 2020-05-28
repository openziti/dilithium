package westworld

import (
	"github.com/michaelquigley/dilithium/protocol/westworld/wb"
	"github.com/michaelquigley/dilithium/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"net"
	"time"
)

type listenerConn struct {
	conn     *net.UDPConn
	peer     *net.UDPAddr
	rxQueue  chan *wb.WireMessage
	seq      *util.Sequence
	txWindow *txWindow
	rxWindow *rxWindow
	pool     *wb.BufferPool
}

func newListenerConn(conn *net.UDPConn, peer *net.UDPAddr) *listenerConn {
	lc := &listenerConn{
		conn:    conn,
		peer:    peer,
		rxQueue: make(chan *wb.WireMessage, 1024),
		seq:     util.NewSequence(0),
		pool:    wb.NewBufferPool("listenerConn"),
	}
	ackQueue := make(chan int32, queueLength)
	lc.txWindow = newTxWindow(ackQueue, conn, peer)
	lc.rxWindow = newRxWindow(ackQueue, conn, peer, lc.txWindow)
	return lc
}

func (self *listenerConn) Read(p []byte) (int, error) {
	if n, err := self.rxWindow.read(p); err == nil && n > 0 {
		//logrus.Infof("+[%d] <-", n)
		return n, nil
	}

	for {
		wm, ok := <-self.rxQueue
		if !ok {
			return 0, errors.New("closed")
		}

		if wm.Type == wb.DATA {
			//logrus.Infof("<- {#%d,@%d}[%d] <-", wm.Sequence, wm.Ack, len(wm.Data))

			if wm.Ack != -1 {
				self.txWindow.ack(wm.Ack)
			}

			if err := self.rxWindow.rx(wm); err != nil {
				logrus.Errorf("rxWindow.rx (%v)", err)
				return 0, errors.Wrap(err, "rxWindow.rx")
			}

			if n, err := self.rxWindow.read(p); err == nil && n > 0 {
				//logrus.Infof("[%d] <-", n)
				return n, nil
			}

		} else if wm.Type == wb.ACK {
			//logrus.Infof("<- {@%d} <-", wm.Ack)

			if wm.Ack != -1 {
				self.txWindow.ack(wm.Ack)
			}
			wm.Free("listenerConn.Read (after ack)")

		} else {
			wm.Free("listenerConn.Read (after invalid message)")
			return 0, errors.Errorf("invalid message [%d]", wm.Type)
		}
	}
}

func (self *listenerConn) Write(p []byte) (int, error) {
	wm, err := wb.NewData(self.seq.Next(), p, self.pool)
	if err != nil {
		return 0, errors.Wrap(err, "alloc")
	}
	self.txWindow.tx(wm)

	select {
	case err := <-self.txWindow.txErrors:
		return 0, err
	default:
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

func (self *listenerConn) queue(wm *wb.WireMessage) {
	self.rxQueue <- wm
}

func (self *listenerConn) hello(hello *wb.WireMessage) error {
	logrus.Infof("{hello} <- [%s]", self.peer)

	self.rxWindow.accepted = hello.Sequence
	hello.Free("listenerConn.hello (after hello receive)")

	helloAckSeq := self.seq.Next()

	wm, err := wb.NewHelloAck(helloAckSeq, hello.Sequence, self.pool)
	if err != nil {
		return errors.Wrap(err, "alloc")
	}
	if err := wm.WriteMessage(self.conn, self.peer); err != nil {
		wm.Free("listenerConn.hello (after hello ack failed)")
		return errors.Wrap(err, "write hello ack")
	}
	wm.Free("listenerConn.hello (after hello ack)")
	logrus.Infof("{helloack} -> [%s]", self.peer)

	select {
	case ack, ok := <-self.rxQueue:
		if !ok {
			return errors.New("rx queue closed")
		}
		ack.Free("listenerConn.hello (after ack received)")

		if ack.Type == wb.ACK && ack.Ack == helloAckSeq {
			logrus.Infof("{ack} <- [%s]", self.peer)
			logrus.Infof("connection established with [%s]", self.peer)
			return nil
		}
		return errors.New("invalid hello ack")

	case <-time.After(5 * time.Second):
		return errors.New("timeout")
	}
}
