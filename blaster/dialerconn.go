package blaster

import (
	"encoding/gob"
	"github.com/michaelquigley/dilithium/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"net"
	"time"
)

type dialerConn struct {
	cconn *net.TCPConn
	cenc  *gob.Encoder
	cdec  *gob.Decoder
	dconn *net.UDPConn
	dpeer *net.UDPAddr
	seq   *util.Sequence
}

func newDialerConn(cconn *net.TCPConn, dconn *net.UDPConn, dpeer *net.UDPAddr) *dialerConn {
	return &dialerConn{
		cconn: cconn,
		cenc:  gob.NewEncoder(cconn),
		cdec:  gob.NewDecoder(cconn),
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

func (self *dialerConn) hello() error {
	if err := self.cenc.Encode(&cmsg{self.seq.Next(), Sync}); err != nil {
		_ = self.cconn.Close()
		_ = self.dconn.Close()
		return errors.Wrap(err, "encode sync")
	}

	rplMsg := cmsg{}
	if err := self.cdec.Decode(&rplMsg); err != nil {
		_ = self.cconn.Close()
		_ = self.dconn.Close()
		return errors.Wrap(err, "decode cmsg")
	}
	if rplMsg.Mt != Hello {
		_ = self.cconn.Close()
		_ = self.dconn.Close()
		return errors.Errorf("expected hello, Mt [%d]", rplMsg.Mt)
	}
	rplHello := chello{}
	if err := self.cdec.Decode(&rplHello); err != nil {
		_ = self.cconn.Close()
		_ = self.dconn.Close()
		return errors.Wrap(err, "decode chello")
	}
	logrus.Infof("session nonce [%s]", rplHello.Nonce)

	_ = self.cconn.Close()
	_ = self.dconn.Close()
	return errors.New("not implemented")
}
