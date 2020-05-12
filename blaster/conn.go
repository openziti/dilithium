package blaster

import (
	"net"
	"time"
)

func newConn(cconn net.Conn) *conn {
	return &conn{cconn: cconn}
}

type conn struct {
	cconn net.Conn
}

func (self *conn) Read(p []byte) (n int, err error) {
	return
}

func (self *conn) Write(p []byte) (n int, err error) {
	return
}

func (self *conn) Close() error {
	return self.cconn.Close()
}

func (self *conn) RemoteAddr() net.Addr {
	return self.cconn.RemoteAddr()
}

func (self *conn) LocalAddr() net.Addr {
	return self.cconn.LocalAddr()
}

func (self *conn) SetDeadline(t time.Time) error {
	return nil
}

func (self *conn) SetReadDeadline(t time.Time) error {
	return nil
}

func (self *conn) SetWriteDeadline(t time.Time) error {
	return nil
}

func (self *conn) hello() error {
	return nil
}
