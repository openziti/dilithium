package westworld

import (
	"github.com/michaelquigley/dilithium/protocol/westworld/wb"
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
	rxPool   *wb.BufferPool
	txPool   *wb.BufferPool
}

func newDialerConn(conn *net.UDPConn, peer *net.UDPAddr) *dialerConn {
	dc := &dialerConn{
		conn: conn,
		peer: peer,
		seq:  util.NewSequence(0),
		rxPool: wb.NewBufferPool("dialerConn.Read"),
		txPool: wb.NewBufferPool("dialerConn.Write"),
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
		wm, _, err := wb.ReadWireMessage(self.conn, self.rxPool)
		if err != nil {
			return 0, errors.Wrap(err, "read")
		}
		wm.Ref()

		if wm.Type == wb.DATA {
			wm.Touch()
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
			wm.Touch()
			//logrus.Infof("<- {@%d} <-", wm.Ack)

			if wm.Ack != -1 {
				self.txWindow.ack(wm.Ack)
			}
			wm.Unref()

		} else {
			wm.Unref()
			return 0, errors.Errorf("invalid message [%d]", wm.Type)
		}
	}
}

func (self *dialerConn) Write(p []byte) (int, error) {
	wm, err := wb.NewData(self.seq.Next(), p, self.txPool)
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
	wm, err := wb.NewHello(helloSeq, self.txPool)
	if err != nil {
		return errors.Wrap(err, "alloc")
	}
	wm.Ref()
	defer wm.Unref()
	if err := wm.WriteMessage(self.conn, self.peer); err != nil {
		return errors.Wrap(err, "write hello")
	}
	logrus.Infof("{hello} -> [%s]", self.peer)

	if err := self.conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return errors.Wrap(err, "set read deadline")
	}
	wm, _, err = wb.ReadWireMessage(self.conn, self.rxPool)
	if err != nil {
		return errors.Wrap(err, "read hello ack")
	}
	wm.Ref()
	defer wm.Unref()

	if wm.Type != wb.HELLO {
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

	wm, err = wb.NewAck(wm.Sequence, self.txPool)
	if err != nil {
		return errors.Wrap(err, "alloc")
	}
	wm.Ref()
	defer wm.Unref()

	if err := wm.WriteMessage(self.conn, self.peer); err != nil {
		return errors.Wrap(err, "write ack")
	}
	logrus.Infof("{ack} -> [%s]", self.peer)

	logrus.Infof("connection established with [%s]", self.peer)

	return nil
}
