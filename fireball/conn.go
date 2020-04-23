package fireball

import (
	"net"
	"time"
)

type listenerConn struct {
	conn  *net.UDPConn
	local *net.UDPAddr
	peer  *net.UDPAddr
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
	return self.peer
}

func (self *listenerConn) LocalAddr() net.Addr {
	return self.local
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

func (self *listenerConn) queue(m *message) {
}

type conn struct {
	conn  *net.UDPConn
	local *net.UDPAddr
	peer  *net.UDPAddr
}

func (self *conn) send(m *message) error {
	return nil
}
