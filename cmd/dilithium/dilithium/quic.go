package dilithium

import (
	"context"
	"github.com/lucas-clemente/quic-go"
	"net"
	"time"
)

type quicAccepter struct {
	listener quic.Listener
}

func (self *quicAccepter) Accept() (net.Conn, error) {
	session, err := self.listener.Accept(context.Background())
	if err != nil {
		return nil, err
	}
	stream, err := session.AcceptStream(context.Background())
	if err != nil {
		return nil, err
	}
	return &quicConn{session, stream}, nil
}

type quicConn struct {
	session quic.Session
	stream  quic.Stream
}

func (self *quicConn) Read(p []byte) (int, error) {
	return self.stream.Read(p)
}

func (self *quicConn) Write(p []byte) (int, error) {
	return self.stream.Write(p)
}

func (self *quicConn) Close() error {
	e1 := self.stream.Close()
	if e1 != nil {
		return e1
	}
	e2 := self.stream.Close()
	if e2 != nil {
		return e2
	}
	return nil
}

func (self *quicConn) LocalAddr() net.Addr {
	return nil
}

func (self *quicConn) RemoteAddr() net.Addr {
	return nil
}

func (self *quicConn) SetDeadline(t time.Time) error {
	return nil
}

func (self *quicConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (self *quicConn) SetWriteDeadline(t time.Time) error {
	return nil
}
