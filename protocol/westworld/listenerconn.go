package westworld

import (
	"github.com/michaelquigley/dilithium/protocol/westworld/pb"
	"github.com/michaelquigley/dilithium/util"
	"github.com/pkg/errors"
	"net"
	"time"
)

type listenerConn struct {
	conn    *net.UDPConn
	peer    *net.UDPAddr
	rxQueue chan *pb.WireMessage
	seq     *util.Sequence
}

func newListenerConn(conn *net.UDPConn, peer *net.UDPAddr) *listenerConn {
	return &listenerConn{
		conn:    conn,
		peer:    peer,
		rxQueue: make(chan *pb.WireMessage, 1024),
		seq:     util.NewSequence(util.RandomSequence()),
	}
}

func (self *listenerConn) Read(p []byte) (n int, err error) {
	return 0, nil
}

func (self *listenerConn) Write(p []byte) (n int, err error) {
	return 0, nil
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
	helloAckSeq := self.seq.Next()
	if err := pb.WriteWireMessage(pb.NewHelloAck(helloAckSeq, hello.Sequence), self.conn, self.peer); err != nil {
		return errors.Wrap(err, "write hello ack")
	}
	select {
	case ack, ok := <-self.rxQueue:
		if !ok {
			return errors.New("rx queue closed")
		}
		if ack.Type == pb.MessageType_ACK && ack.Ack == helloAckSeq {
			return nil
		}
		return errors.New("invalid hello ack")

	case <-time.After(5 * time.Second):
		return errors.New("timeout")
	}
}