package blaster

import (
	"encoding/gob"
	"github.com/michaelquigley/dilithium/blaster/pb"
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
			if wireMessage, err := pb.FromData(buffer[:n]); err == nil {
				if lc, found := self.active[peer.String()]; found {
					lc.rxq <- &pb.WireMessagePeer{WireMessage: wireMessage, Peer: peer}
				} else {
					if wireMessage.Type == pb.MessageType_HELLO {
						if lc, found := self.syncing[wireMessage.HelloPayload.Session]; found {
							lc.rxq <- &pb.WireMessagePeer{WireMessage: wireMessage, Peer: peer}
						}
					} else {
						logrus.Errorf("no recipient for message [%s] at [%s]", wireMessage, peer)
					}
				}
			} else {
				logrus.Errorf("error decoding message (%v)", err)
			}
		} else {
			logrus.Errorf("read error (%v)", err)
		}
	}
}
