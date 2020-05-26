package westworld

import (
	"github.com/michaelquigley/dilithium/protocol/westworld/pb"
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
	txWindow *txWindow
	rxWindow *rxWindow
}

func newDialerConn(conn *net.UDPConn, peer *net.UDPAddr) *dialerConn {
	dc := &dialerConn{
		conn: conn,
		peer: peer,
		seq:  util.NewSequence(0),
	}
	ackQueue := make(chan int32, queueLength)
	dc.txWindow = newTxWindow(ackQueue, conn, peer)
	dc.rxWindow = newRxWindow(ackQueue, conn, peer, dc.txWindow)
	return dc
}

func (self *dialerConn) Read(p []byte) (int, error) {
	if n, err := self.rxWindow.read(p); err == nil && n > 0 {
		//logrus.Infof("+[%d] <-", n)
		return n, nil
	}

	for {
		wm, _, err := pb.ReadWireMessage(self.conn)
		if err != nil {
			return 0, errors.Wrap(err, "read")
		}

		if wm.Type == pb.MessageType_DATA {
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

		} else if wm.Type == pb.MessageType_ACK {
			//logrus.Infof("<- {@%d} <-", wm.Ack)

			if wm.Ack != -1 {
				self.txWindow.ack(wm.Ack)
			}

		} else {
			return 0, errors.Errorf("invalid message [%s]", wm.Type.String())
		}
	}
}

func (self *dialerConn) Write(p []byte) (int, error) {
	wm := pb.NewData(self.seq.Next(), p)
	self.txWindow.tx(wm)

	select {
	case err := <-self.txWindow.txErrors:
		return 0, err
	default:
	}

	return len(p), nil
}

func (self *dialerConn) Close() error {
	return nil
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

func (self *dialerConn) hello() error {
	helloSeq := self.seq.Next()
	if err := pb.WriteWireMessage(pb.NewHello(helloSeq), self.conn, self.peer); err != nil {
		return errors.Wrap(err, "write hello")
	}
	logrus.Infof("{hello} -> [%s]", self.peer)

	if err := self.conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return errors.Wrap(err, "set read deadline")
	}
	wm, _, err := pb.ReadWireMessage(self.conn)
	if err != nil {
		return errors.Wrap(err, "read hello ack")
	}
	if wm.Type != pb.MessageType_HELLO {
		return errors.Wrap(err, "unexpected response")
	}
	if wm.Ack != helloSeq {
		return errors.New("invalid hello ack")
	}
	if err := self.conn.SetReadDeadline(time.Time{}); err != nil {
		return errors.Wrap(err, "clear read deadline")
	}
	logrus.Infof("{helloack} <- [%s]", self.peer)

	self.rxWindow.accepted = wm.Sequence

	if err := pb.WriteWireMessage(pb.NewAck(wm.Sequence), self.conn, self.peer); err != nil {
		return errors.Wrap(err, "write ack")
	}
	logrus.Infof("{ack} -> [%s]", self.peer)

	logrus.Infof("connection established with [%s]", self.peer)

	return nil
}
