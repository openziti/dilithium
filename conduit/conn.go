package conduit

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

type dialerConn struct {
	conn  *net.UDPConn
	local *net.UDPAddr
	peer  *net.UDPAddr
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
	return self.peer
}

func (self *dialerConn) LocalAddr() net.Addr {
	return self.local
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
