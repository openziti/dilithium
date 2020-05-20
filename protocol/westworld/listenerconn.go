package westworld

import (
	"github.com/michaelquigley/dilithium/protocol/westworld/pb"
	"github.com/michaelquigley/dilithium/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"net"
	"time"
)

type listenerConn struct {
	conn     *net.UDPConn
	peer     *net.UDPAddr
	rxQueue  chan *pb.WireMessage
	seq      *util.Sequence
	txWindow *txWindow
	rxWindow *rxWindow
}

func newListenerConn(conn *net.UDPConn, peer *net.UDPAddr) *listenerConn {
	lc := &listenerConn{
		conn:    conn,
		peer:    peer,
		rxQueue: make(chan *pb.WireMessage, 1024),
		seq:     util.NewSequence(util.RandomSequence()),
	}
	ackQueue := make(chan int32, ackQueueLength)
	ackSnoozer := make(chan int32, ackQueueLength)
	lc.txWindow = newTxWindow(ackQueue, ackSnoozer, conn, peer)
	lc.rxWindow = newRxWindow(ackQueue, ackSnoozer, conn, peer, lc.txWindow)
	return lc
}

func (self *listenerConn) Read(p []byte) (int, error) {
	if n, err := self.rxWindow.read(p); err == nil && n > 0 {
		logrus.Infof("+[%d] <-", n)
		return n, nil
	}

	for {
		wm, ok := <- self.rxQueue
		if !ok {
			return 0, errors.New("closed")
		}

		if wm.Type == pb.MessageType_DATA {
			logrus.Infof("<- {#%d,@%d}[%d] <-", wm.Sequence, wm.Ack, len(wm.Data))

			if wm.Ack != -1 {
				self.txWindow.ack(wm.Ack)
			}

			if err := self.rxWindow.rx(wm); err != nil {
				logrus.Errorf("rxWindow.rx (%v)", err)
				return 0, errors.Wrap(err, "rxWindow.rx")
			}

			if n, err := self.rxWindow.read(p); err == nil && n > 0 {
				logrus.Infof("[%d] <-", n)
				return n, nil
			} else {
				if err != nil {
					logrus.Errorf("read (%v)", err)
				}
			}

		} else if wm.Type == pb.MessageType_ACK {
			logrus.Infof("<- {@%d} <-", wm.Ack)

			if wm.Ack != -1 {
				self.txWindow.ack(wm.Ack)
			}

		} else {
			return 0, errors.Errorf("invalid message [%s]", wm.Type.String())
		}
	}
}

func (self *listenerConn) Write(p []byte) (int, error) {
	wm := pb.NewData(self.seq.Next(), p)
	self.txWindow.tx(wm)
	if err := pb.WriteWireMessage(wm, self.conn, self.peer); err != nil {
		return 0, errors.Wrap(err, "write")
	}
	logrus.Infof("[%d] -> {#%d,@%d}[%d] ->", len(p), wm.Sequence, wm.Ack, len(wm.Data))
	return len(wm.Data), nil
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

func (self *listenerConn) queue(wm *pb.WireMessage) {
	self.rxQueue <- wm
}

func (self *listenerConn) hello(hello *pb.WireMessage) error {
	logrus.Infof("{hello} <- [%s]", self.peer)

	self.rxWindow.accepted = hello.Sequence

	helloAckSeq := self.seq.Next()
	if err := pb.WriteWireMessage(pb.NewHelloAck(helloAckSeq, hello.Sequence), self.conn, self.peer); err != nil {
		return errors.Wrap(err, "write hello ack")
	}
	logrus.Infof("{helloack} -> [%s]", self.peer)

	select {
	case ack, ok := <-self.rxQueue:
		if !ok {
			return errors.New("rx queue closed")
		}
		if ack.Type == pb.MessageType_ACK && ack.Ack == helloAckSeq {
			logrus.Infof("{ack} <- [%s]", self.peer)
			logrus.Infof("connection established with [%s]", self.peer)
			return nil
		}
		return errors.New("invalid hello ack")

	case <-time.After(5 * time.Second):
		return errors.New("timeout")
	}
}
