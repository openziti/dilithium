package blaster

import (
	"github.com/pkg/errors"
	"net"
)

func Dial(address string) (net.Conn, error) {
	dConn, err := net.ListenUDP("udp", nil)
	if err != nil {
		return nil, errors.Wrap(err, "listen dConn")
	}
	dPeer, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return nil, errors.Wrap(err, "resolve dPeer")
	}

	cAddr, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		return nil, errors.Wrap(err, "resolve cAddr")
	}
	cConn, err := net.DialTCP("tcp", nil, cAddr)
	if err != nil {
		return nil, errors.Wrap(err, "dial cConn")
	}

	conn := newDialerConn(cConn, dConn, dPeer)
	if err := conn.hello(); err != nil {
		return nil, errors.Wrap(err, "hello")
	}

	return conn, nil
}