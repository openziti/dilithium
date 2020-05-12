package blaster

import (
	"encoding/gob"
	"github.com/michaelquigley/dilithium/util"
	"github.com/sirupsen/logrus"
	"net"
	"time"
)

type listenerConn struct {
	listn *listener
	cconn net.Conn
	cenc  *gob.Encoder
	cdec  *gob.Decoder
	dconn *net.UDPConn
	dpeer *net.UDPAddr
	rxq   chan *cmsgPair
	seq   *util.Sequence
}

func newListenerConn(listn *listener, cconn net.Conn, dconn *net.UDPConn) *listenerConn {
	return &listenerConn{
		listn: listn,
		cconn: cconn,
		cenc:  gob.NewEncoder(cconn),
		cdec:  gob.NewDecoder(cconn),
		dconn: dconn,
		rxq:   make(chan *cmsgPair, 1024),
		seq:   util.NewSequence(),
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
	return self.cconn.RemoteAddr()
}

func (self *listenerConn) LocalAddr() net.Addr {
	return self.cconn.LocalAddr()
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

func (self *listenerConn) queue(p *cmsgPair) {
	self.rxq <- p
}

func (self *listenerConn) hello() {
	req := cmsg{}
	if err := self.cdec.Decode(&req); err == nil {
		logrus.Infof("received hello request")
	} else {
		logrus.Errorf("error decoding hello request (%v)", err)
		return
	}
	if err := self.cenc.Encode(&cmsg{self.seq.Next(), Hello}); err != nil {
		logrus.Infof("sent hello response")
	} else {
		logrus.Errorf("error encoding hello response (%v)", err)
	}
	// receive dconn hello
	// transmit dconn hello
	// wait for cconn ok
}
