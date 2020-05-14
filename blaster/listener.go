package blaster

import (
	"github.com/michaelquigley/dilithium/blaster/pb"
	"github.com/michaelquigley/dilithium/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"net"
)

func Listen(cAddr *net.TCPAddr, dAddr *net.UDPAddr) (net.Listener, error) {
	cListener, err := net.ListenTCP("tcp", cAddr)
	if err != nil {
		return nil, errors.Wrap(err, "cListener listen")
	}
	dConn, err := net.ListenUDP("udp", dAddr)
	if err != nil {
		return nil, errors.Wrap(err, "bind udp")
	}
	l := &listener{
		cListener: cListener,
		dConn:     dConn,
		connected: make(map[string]*listenerConn),
		syncing:   make(map[string]*listenerConn),
	}
	go l.rxer()
	return l, nil
}

type listener struct {
	cListener *net.TCPListener
	dConn     *net.UDPConn
	connected map[string]*listenerConn
	syncing   map[string]*listenerConn
}

func (self *listener) Accept() (net.Conn, error) {
	cConn, err := self.cListener.Accept()
	if err != nil {
		return nil, errors.Wrap(err, "accept cConn")
	}
	session := util.GenerateSessionId()
	conn := newListenerConn(self, session, cConn, self.dConn)
	self.syncing[session] = conn

	if err := conn.hello(); err != nil {
		return nil, errors.Wrap(err, "hello")
	}

	return conn, nil
}

func (self *listener) Close() error {
	return self.cListener.Close()
}

func (self *listener) Addr() net.Addr {
	return self.cListener.Addr()
}

func (self *listener) rxer() {
	logrus.Info("started")
	defer logrus.Warnf("exited")

	buffer := make([]byte, 64*1024)
	for {
		if n, peer, err := self.dConn.ReadFromUDP(buffer); err == nil {
			if wireMessage, err := pb.FromData(buffer[:n]); err == nil {
				if lc, found := self.connected[peer.String()]; found {
					lc.dRxQueue <- &pb.AddressedWireMessage{WireMessage: wireMessage, FromPeer: peer}
				} else {
					if wireMessage.Type == pb.MessageType_HELLO {
						if lc, found := self.syncing[wireMessage.HelloPayload.Session]; found {
							lc.dRxQueue <- &pb.AddressedWireMessage{WireMessage: wireMessage, FromPeer: peer}
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
