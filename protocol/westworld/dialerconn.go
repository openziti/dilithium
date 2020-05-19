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
	conn   *net.UDPConn
	peer   *net.UDPAddr
	seq    *util.Sequence
}

func newDialerConn(conn *net.UDPConn, peer *net.UDPAddr) *dialerConn {
	return &dialerConn{
		conn:   conn,
		peer:   peer,
		seq:    util.NewSequence(util.RandomSequence()),
	}
}

func (self *dialerConn) Read(p []byte) (int, error) {
	wm, _, err := pb.ReadWireMessage(self.conn)
	if err != nil {
		return 0, errors.Wrap(err, "read")
	}
	if wm.Type == pb.MessageType_DATA {
		n := copy(p, wm.Data)
		logrus.Infof("[%d] <- {#%d}[%d] <-", n, wm.Sequence, len(wm.Data))
		return n, nil
	}
	return 0, errors.New("invalid message")
}

func (self *dialerConn) Write(p []byte) (int, error) {
	wm := pb.NewData(self.seq.Next(), p)
	if err := pb.WriteWireMessage(wm, self.conn, self.peer); err != nil {
		return 0, errors.Wrap(err, "write")
	}
	logrus.Infof("[%d] -> {#%d}[%d] ->", len(p), wm.Sequence, len(wm.Data))
	return len(wm.Data), nil
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

	if err := pb.WriteWireMessage(pb.NewAck(wm.Sequence), self.conn, self.peer); err != nil {
		return errors.Wrap(err, "write ack")
	}
	logrus.Infof("{ack} -> [%s]", self.peer)

	logrus.Infof("connection established with [%s]", self.peer)

	return nil
}
