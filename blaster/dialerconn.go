package blaster

import (
	"github.com/michaelquigley/dilithium/util"
	"net"
	"time"
)

type dialerConn struct {
	cconn *net.TCPConn
	dconn *net.UDPConn
	dpeer *net.UDPAddr
	seq   *util.Sequence
}

func newDialerConn(cconn *net.TCPConn, dconn *net.UDPConn, dpeer *net.UDPAddr) *dialerConn {
	return &dialerConn{
		cconn: cconn,
		dconn: dconn,
		dpeer: dpeer,
		seq:   util.NewSequence(),
	}
}

func (self *dialerConn) Read(p []byte) (n int, err error) {
	return
}

func (self *dialerConn) Write(p []byte) (n int, err error) {
	return
}

func (self *dialerConn) Close() error {
	return nil
}

func (self *dialerConn) RemoteAddr() net.Addr {
	return self.cconn.RemoteAddr()
}

func (self *dialerConn) LocalAddr() net.Addr {
	return self.cconn.LocalAddr()
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