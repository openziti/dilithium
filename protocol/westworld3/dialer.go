package westworld3

import (
	"github.com/pkg/errors"
	"net"
)

func Dial(addr *net.UDPAddr, profileId byte) (conn net.Conn, err error) {
	profile, found := profileRegistry[profileId]
	if !found {
		return nil, errors.Errorf("no profile [%d]", profileId)
	}

	var lConn *net.UDPConn
	lConn, err = net.ListenUDP("udp", nil)
	if err != nil {
		return nil, errors.Wrap(err, "listen")
	}
	if err := lConn.SetReadBuffer(profile.RxBufferSz); err != nil {
		return nil, errors.Wrap(err, "rx buffer")
	}
	if err := lConn.SetWriteBuffer(profile.TxBufferSz); err != nil {
		return nil, errors.Wrap(err, "tx buffer")
	}

	var dConn *dialerConn
	dConn, err = newDialerConn(lConn, addr, profile)
	if err != nil {
		return nil, errors.Wrap(err, "create dialer conn")
	}
	if err = dConn.hello(); err != nil {
		return nil, errors.Wrap(err, "hello")
	}

	return dConn, nil
}
