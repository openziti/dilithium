package blaster

import (
	"github.com/pkg/errors"
	"net"
)

func Dial(address string) (net.Conn, error) {
	dconn, err := net.ListenUDP("udp", nil)
	if err != nil {
		return nil, errors.Wrap(err, "listen dconn")
	}
	dpeer, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return nil, errors.Wrap(err, "resolve dpeer")
	}

	caddr, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		return nil, errors.Wrap(err, "resolve caddr")
	}
	cconn, err := net.DialTCP("tcp", nil, caddr)
	if err != nil {
		return nil, errors.Wrap(err, "dial caddr")
	}

	dc := newDialerConn(cconn, dconn, dpeer)
	if err := dc.hello(); err != nil {
		return nil, errors.Wrap(err, "hello")
	}

	return dc, nil
}