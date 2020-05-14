package blaster

import (
	"github.com/michaelquigley/dilithium/blaster/pb"
	"github.com/michaelquigley/dilithium/util"
	"net"
	"time"
)

type listenerConn struct {
	cListener *listener
	session   string
	cConn     net.Conn
	cSeq      *util.Sequence
	dConn     *net.UDPConn
	dPeer     *net.UDPAddr
	dSeq      *util.Sequence
	dRxQueue  chan *pb.AddressedWireMessage
}

func newListenerConn(cListener *listener, session string, cConn net.Conn, dConn *net.UDPConn) *listenerConn {
	return &listenerConn{
		cListener: cListener,
		session:   session,
		cConn:     cConn,
		cSeq:      util.NewSequence(),
		dConn:     dConn,
		dSeq:      util.NewSequence(),
		dRxQueue:  make(chan *pb.AddressedWireMessage, 1024),
	}
}

func (self *listenerConn) Read(p []byte) (n int, err error) {
	return
}

func (self *listenerConn) Write(p []byte) (n int, err error) {
	return
}

func (self *listenerConn) Close() error {
	return nil
}

func (self *listenerConn) RemoteAddr() net.Addr {
	return self.cConn.RemoteAddr()
}

func (self *listenerConn) LocalAddr() net.Addr {
	return self.cConn.LocalAddr()
}

func (self *listenerConn) SetDeadline(t time.Time) error {
	return nil
}

func (self *listenerConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (self *listenerConn) SetWriteDeadline(t time.Time) error {
	return nil
}

func (self *listenerConn) queue(awm *pb.AddressedWireMessage) {
	self.dRxQueue <- awm
}
