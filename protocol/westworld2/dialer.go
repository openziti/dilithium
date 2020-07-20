package westworld2

import (
	"github.com/pkg/errors"
	"net"
)

func Dial(addr *net.UDPAddr, config *Config) (conn net.Conn, err error) {
	var lConn *net.UDPConn
	lConn, err = net.ListenUDP("udp", nil)
	if err != nil {
		return nil, errors.Wrap(err, "listen")
	}
	if err := lConn.SetReadBuffer(config.rxBufferSz); err != nil {
		return nil, errors.Wrap(err, "rx buffer")
	}
	if err := lConn.SetWriteBuffer(config.txBufferSz); err != nil {
		return nil, errors.Wrap(err, "tx buffer")
	}

	dConn := newDialerConn(lConn, addr, config)
	if err = dConn.hello(); err != nil {
		return nil, errors.Wrap(err, "hello")
	}

	return dConn, nil
}
