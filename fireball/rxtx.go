package fireball

import (
	"github.com/pkg/errors"
	"net"
)

func readMessage(conn *net.UDPConn) (*message, *net.UDPAddr, error) {
	data := make([]byte, 65507)
	if n, peer, err := conn.ReadFromUDP(data); err == nil {
		if m, err := unmarshal(data[:n]); err == nil {
			return m, peer, nil
		} else {
			return nil, nil, errors.Wrap(err, "unmarshal")
		}
	} else {
		return nil, nil, errors.Wrap(err, "read")
	}
}