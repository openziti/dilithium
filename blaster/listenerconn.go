package blaster

import (
	"github.com/michaelquigley/dilithium/util"
	"net"
	"time"
)

type listenerConn struct {
	cconn *net.TCPConn
	dconn *net.UDPConn
	dpeer *net.UDPAddr
	rxq   chan *cmsgPair
	seq   *util.Sequence
}

func newListenerConn(cconn *net.TCPConn, dconn *net.UDPConn, dpeer *net.UDPAddr) *listenerConn {
	return &listenerConn{
		cconn: cconn,
		dconn: dconn,
		dpeer: dpeer,
		rxq:   make(chan *cmsgPair, 1024),
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

func (self *listenerConn) queue(p *cmsgPair) {
	self.rxq <- p
}