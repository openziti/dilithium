package blaster

import (
	"bytes"
	"encoding/gob"
	"github.com/michaelquigley/dilithium/util"
	"github.com/pkg/errors"
	"net"
	"time"
)

type dialerConn struct {
	sessn string
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

func (self *dialerConn) readCmsgPair() (*cmsgPair, error) {
	buffer := make([]byte, 64*1024)
	n, peer, err := self.dconn.ReadFromUDP(buffer)
	if err != nil {
		return nil, errors.Wrap(err, "read error")
	}

	data := bytes.NewBuffer(buffer[:n])
	dec := gob.NewDecoder(data)
	mh := cmsg{}
	if err := dec.Decode(&mh); err != nil {
		return nil, errors.Wrap(err, "decode cmsg")
	}
	cp, err := decode(mh, dec)
	if err != nil {
		return nil, errors.Wrap(err, "decode error")
	}
	cp.peer = peer

	return cp, nil
}
