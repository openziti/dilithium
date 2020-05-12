package blaster

import (
	"github.com/pkg/errors"
	"net"
)

func Listen(caddr *net.TCPAddr, daddr *net.UDPAddr) (net.Listener, error) {
	clistener, err := net.ListenTCP("tcp", caddr)
	if err != nil {
		return nil, errors.Wrap(err, "listen tcp")
	}
	dconn, err := net.ListenUDP("udp", daddr)
	if err != nil {
		return nil, errors.Wrap(err, "bind udp")
	}
	l := &listener{
		clistener: clistener,
		dconn:     dconn,
	}
	return l, nil
}

type listener struct {
	clistener *net.TCPListener
	dconn     *net.UDPConn
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
