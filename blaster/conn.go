package blaster

import (
	"bytes"
	"encoding/gob"
	"github.com/michaelquigley/dilithium/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"net"
	"time"
)

func newConn(cconn net.Conn) *conn {
	c := &conn{
		cconn: cconn,
		seq:   util.NewSequence(),
		rxq:   make(chan cmsgPair, 1024),
	}
	c.cenc = gob.NewEncoder(c.cconn)
	c.cdec = gob.NewDecoder(c.cconn)
	return c
}

type conn struct {
	cconn net.Conn
	cenc  *gob.Encoder
	cdec  *gob.Decoder
	dconn *net.UDPConn
	seq   *util.Sequence
	rxq   chan cmsgPair
}

func (self *conn) Read(p []byte) (n int, err error) {
	return
}

func (self *conn) Write(p []byte) (n int, err error) {
	return
}

func (self *conn) Close() error {
	return self.cconn.Close()
}

func (self *conn) RemoteAddr() net.Addr {
	return self.cconn.RemoteAddr()
}

func (self *conn) LocalAddr() net.Addr {
	return self.cconn.LocalAddr()
}

func (self *conn) SetDeadline(t time.Time) error {
	return nil
}

func (self *conn) SetReadDeadline(t time.Time) error {
	return nil
}

func (self *conn) SetWriteDeadline(t time.Time) error {
	return nil
}

func (self *conn) hello() (err error) {
	err = self.startDconn()
	if err != nil {
		return errors.Wrap(err, "start dconn")
	}

	err = self.cenc.Encode(&cmsg{seq: self.seq.Next(), mt: Hello})
	if err != nil {
		return errors.Wrap(err, "encode hello")
	}

	var inHello cmsg
	err = self.cdec.Decode(&inHello)
	if err != nil {
		return errors.Wrap(err, "decode hello")
	}

	return nil
}

func (self *conn) startDconn() (err error) {
	var daddr *net.UDPAddr
	daddr, err = net.ResolveUDPAddr("udp", self.cconn.LocalAddr().String())
	if err != nil {
		return errors.Wrap(err, "daddr resolution")
	}

	self.dconn, err = net.ListenUDP("udp", daddr)
	if err != nil {
		return errors.Wrap(err, "dconn listen")
	}
	go self.dlisten()

	return nil
}

func (self *conn) dlisten() {
	data := make([]byte, 65507)
	if n, _, err := self.dconn.ReadFromUDP(data); err == nil {
		buffer := bytes.NewBuffer(data[:n])
		dec := gob.NewDecoder(buffer)
		v := cmsg{}
		if err := dec.Decode(&v); err == nil {
			switch v.mt {
			case Hello:
				self.rxq <- cmsgPair{h: v}
			default:
				logrus.Warnf("dropping mt [%d]", v.mt)
			}
		} else {
			logrus.Errorf("error decoding message (%v)", err)
		}
	} else {
		logrus.Errorf("error reading (%v)", err)
	}
}
