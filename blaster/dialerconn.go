package blaster

import (
	"github.com/michaelquigley/dilithium/blaster/pb"
	"github.com/michaelquigley/dilithium/util"
	"github.com/pkg/errors"
	"net"
	"time"
)

type dialerConn struct {
	sessn string
	cconn *net.TCPConn
	dconn *net.UDPConn
	dpeer *net.UDPAddr
	seq   *util.Sequence
}

func newDialerConn(cconn *net.TCPConn, dconn *net.UDPConn, dpeer *net.UDPAddr) *dialerConn {
	return &dialerConn{
		cconn: cconn,
		dconn: dconn,
		dpeer: dpeer,
		seq:   util.NewSequence(),
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
	return self.cconn.RemoteAddr()
}

func (self *dialerConn) LocalAddr() net.Addr {
	return self.cconn.LocalAddr()
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

func (self *dialerConn) readWireMessagePeer() (*pb.WireMessagePeer, error) {
	buffer := make([]byte, 64*1024)
	n, peer, err := self.dconn.ReadFromUDP(buffer)
	if err != nil {
		return nil, errors.Wrap(err, "read error")
	}

	wm, err := pb.FromData(buffer[:n])
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal")
	}

	return &pb.WireMessagePeer{WireMessage: wm, Peer: peer}, nil
}
