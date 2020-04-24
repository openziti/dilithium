package fireball

import (
	"github.com/pkg/errors"
	"net"
)

func Dial(addr *net.UDPAddr) (net.Conn, error) {
	conn, err := net.ListenUDP("udp", nil)
	if err != nil {
		return nil, errors.Wrap(err, "listen")
	}

	hello, err := newHelloMessage().marshal()
	if err != nil {
		return nil, errors.Wrap(err, "marshal")
	}

	n, err := conn.WriteTo(hello, addr)
	if err != nil {
		return nil, errors.Wrap(err, "write")
	}
	if n != len(hello) {
		return nil, errors.New("short write")
	}

	m, peer, err := readMessage(conn)
	if err != nil {
		return nil, errors.Wrap(err, "read")
	}
	if peer.IP.String() != addr.IP.String() || peer.Port != addr.Port {
		return nil, errors.New("peer mismatch")
	}

	if m.message == Hello {
		local, err := net.ResolveUDPAddr("udp", conn.LocalAddr().String())
		if err != nil {
			return nil, errors.Wrap(err, "resolve address")
		}
		dConn := &dialerConn{
			conn:  conn,
			local: local,
			peer:  addr,
		}
		return dConn, nil
		
	} else {
		return nil, errors.New("invalid message received")
	}
}
