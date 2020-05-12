package blaster

import (
	"github.com/pkg/errors"
	"net"
)

func Listen(addr *net.TCPAddr) (net.Listener, error) {
	ltcp, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return nil, errors.Wrap(err, "listen tcp")
	}
	l := &listener{clistener: ltcp}
	return l, nil
}

type listener struct {
	clistener *net.TCPListener
}

func (self *listener) Accept() (net.Conn, error) {
	cconn, err := self.clistener.Accept()
	if err != nil {
		return nil, errors.Wrap(err, "accept")
	}
	c := newConn(cconn)
	if err := c.hello(); err != nil {
		return nil, errors.Wrap(err, "connection setup")
	}
	return c, nil
}

func (self *listener) Close() error {
	return self.clistener.Close()
}

func (self *listener) Addr() net.Addr {
	return self.clistener.Addr()
}