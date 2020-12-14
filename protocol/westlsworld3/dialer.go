package westlsworld3

import (
	"crypto/tls"
	"github.com/openziti/dilithium/protocol/westworld3"
	"net"
)

func Dial(addr *net.UDPAddr, tlsConfig *tls.Config, profileId byte) (net.Conn, error) {
	w3Conn, err := westworld3.Dial(addr, profileId)
	if err != nil {
		return nil, err
	}
	return tls.Client(w3Conn, tlsConfig), nil
}