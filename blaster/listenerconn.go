package blaster

import (
	"github.com/michaelquigley/dilithium/blaster/pb"
	"github.com/michaelquigley/dilithium/util"
	"net"
	"time"
)

type listenerConn struct {
	listn *listener
	sessn string
	cconn net.Conn
	dconn *net.UDPConn
	dpeer *net.UDPAddr
	rxq   chan *pb.WireMessagePeer
	seq   *util.Sequence
}

func newListenerConn(listn *listener, sessn string, cconn net.Conn, dconn *net.UDPConn) *listenerConn {
	return &listenerConn{
		listn: listn,
		sessn: sessn,
		cconn: cconn,
		dconn: dconn,
		rxq:   make(chan *pb.WireMessagePeer, 1024),
		seq:   util.NewSequence(),
	}
}

func (self *listenerConn) Read(p []byte) (n int, err error) {
	return
}

func (self *listenerConn) Write(p []byte) (n int, err error) {
	return
}

func (self *listenerConn) Close() error {
	return nil
}

func (self *listenerConn) RemoteAddr() net.Addr {
	return self.cconn.RemoteAddr()
}

func (self *listenerConn) LocalAddr() net.Addr {
	return self.cconn.LocalAddr()
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

func (self *listenerConn) queue(p *pb.WireMessagePeer) {
	self.rxq <- p
}
