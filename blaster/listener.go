package blaster

import (
	"bytes"
	"encoding/gob"
	"github.com/michaelquigley/dilithium/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"net"
)

func Listen(caddr *net.TCPAddr, daddr *net.UDPAddr) (net.Listener, error) {
	clistener, err := net.ListenTCP("tcp", caddr)
	if err != nil {
		return nil, errors.Wrap(err, "listen tcp")
	}
	dconn, err := net.ListenUDP("udp", daddr)
	if err != nil {
		return nil, errors.Wrap(err, "bind udp")
	}
	l := &listener{
		clistener: clistener,
		dconn:     dconn,
		active:    make(map[string]*listenerConn),
		syncing:   make(map[string]*listenerConn),
	}
	go l.rxer()
	return l, nil
}

type listener struct {
	clistener *net.TCPListener
	cenc      gob.Encoder
	cdec      gob.Decoder
	dconn     *net.UDPConn
	active    map[string]*listenerConn
	syncing   map[string]*listenerConn
}

func (self *listener) Accept() (net.Conn, error) {
	cconn, err := self.clistener.Accept()
	if err != nil {
		return nil, errors.Wrap(err, "accept")
	}
	sessn := util.GenerateSessionId()
	c := newListenerConn(self, sessn, cconn, self.dconn)
	self.syncing[sessn] = c
	if err := c.hello(); err != nil {
		return nil, errors.Wrap(err, "accept")
	}
	return c, nil
}

func (self *listener) Close() error {
	return self.clistener.Close()
}

func (self *listener) Addr() net.Addr {
	return self.clistener.Addr()
}

func (self *listener) rxer() {
	logrus.Info("started")
	defer logrus.Warnf("exited")

	buffer := make([]byte, 64*1024)
	for {
		if n, peer, err := self.dconn.ReadFromUDP(buffer); err == nil {
			data := bytes.NewBuffer(buffer[:n])
			dec := gob.NewDecoder(data)
			mh := cmsg{}
			if err := dec.Decode(&mh); err == nil {
				if cmp, err := decode(mh, dec); err == nil {
					cmp.peer = peer
					if lc, found := self.active[peer.String()]; found {
						lc.rxq <- cmp
					} else {
						if mh.Mt == Hello {
							if lc, found := self.syncing[cmp.p.(chello).Nonce]; found {
								lc.rxq <- cmp
							} else {
								logrus.Errorf("no inactive peer found for [%s]", cmp.p.(chello).Nonce)
							}
						} else {
							logrus.Errorf("invalid Mt [%d] for inactive peer [%s]", mh.Mt, peer)
						}
					}
				} else {
					logrus.Errorf("pair error (%v)", err)
				}
			} else {
				logrus.Errorf("decode error (%v)", err)
			}
		} else {
			logrus.Errorf("read error (%v)", err)
		}
	}
}
