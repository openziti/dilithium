package blaster

import (
	"github.com/michaelquigley/dilithium/blaster/pb"
	"github.com/michaelquigley/dilithium/util"
	"github.com/pkg/errors"
	"net"
	"time"
)

type dialerConn struct {
	session string
	cConn   *net.TCPConn
	cSeq    *util.Sequence
	dConn   *net.UDPConn
	dPeer   *net.UDPAddr
	dSeq    *util.Sequence
}

func newDialerConn(cConn *net.TCPConn, dConn *net.UDPConn, dPeer *net.UDPAddr) *dialerConn {
	return &dialerConn{
		cConn: cConn,
		cSeq:  util.NewSequence(),
		dConn: dConn,
		dPeer: dPeer,
		dSeq:  util.NewSequence(),
	}
}

func (self *dialerConn) Read(p []byte) (n int, err error) {
	return
}

func (self *dialerConn) Write(p []byte) (n int, err error) {
	return
}

func (self *dialerConn) Close() error {
	return nil
}

func (self *dialerConn) RemoteAddr() net.Addr {
	return self.cConn.RemoteAddr()
}

func (self *dialerConn) LocalAddr() net.Addr {
	return self.cConn.LocalAddr()
}

func (self *dialerConn) SetDeadline(t time.Time) error {
	return nil
}

func (self *dialerConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (self *dialerConn) SetWriteDeadline(t time.Time) error {
	return nil
}

func (self *dialerConn) readWireMessage() (*pb.AddressedWireMessage, error) {
	buffer := make([]byte, 64*1024)
	n, peer, err := self.dConn.ReadFromUDP(buffer)
	if err != nil {
		return nil, errors.Wrap(err, "read error")
	}

	wm, err := pb.FromData(buffer[:n])
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal")
	}

	return &pb.AddressedWireMessage{WireMessage: wm, FromPeer: peer}, nil
}
