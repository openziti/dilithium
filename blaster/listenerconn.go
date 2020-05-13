package blaster

import (
	"encoding/gob"
	"github.com/michaelquigley/dilithium/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"net"
	"time"
)

type listenerConn struct {
	listn *listener
	sessn string
	cconn net.Conn
	cenc  *gob.Encoder
	cdec  *gob.Decoder
	dconn *net.UDPConn
	dpeer *net.UDPAddr
	rxq   chan *cmsgPair
	seq   *util.Sequence
}

func newListenerConn(listn *listener, sessn string, cconn net.Conn, dconn *net.UDPConn) *listenerConn {
	return &listenerConn{
		listn: listn,
		sessn: sessn,
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

func (self *listenerConn) hello() error {
	logrus.Infof("reading sync")
	reqMsg := cmsg{}
	if err := self.cdec.Decode(&reqMsg); err != nil {
		_ = self.cconn.Close()
		return errors.Wrap(err, "decode cmsg")
	}
	if reqMsg.Mt != Sync {
		_ = self.cconn.Close()
		return errors.Errorf("expected sync got mt [%d]", reqMsg.Mt)
	}
	logrus.Infof("got sync")

	logrus.Infof("sending hello")
	if err := self.cenc.Encode(&cmsg{self.seq.Next(), Hello}); err != nil {
		_ = self.cconn.Close()
		return errors.Wrap(err, "encode cmsg")
	}
	if err := self.cenc.Encode(&chello{self.sessn}); err != nil {
		_ = self.cconn.Close()
		return errors.Wrap(err, "encode chello")
	}
	logrus.Infof("sent hello")

	// receive dconn hello
	// transmit dconn hello
	// wait for cconn ok
	return nil
}
