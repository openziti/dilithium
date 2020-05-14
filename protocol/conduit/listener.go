package conduit

import (
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"net"
	"sync"
)

func Listen(addr *net.UDPAddr) (net.Listener, error) {
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, errors.Wrap(err, "listen")
	}
	l := &listener{
		lock:        new(sync.Mutex),
		conn:        conn,
		addr:        addr,
		peers:       make(map[string]*listenerConn),
		acceptQueue: make(chan net.Conn, 1024),
	}
	go l.run()
	return l, nil
}

func (self *listener) Accept() (net.Conn, error) {
	conn, ok := <-self.acceptQueue
	if !ok {
		return nil, errors.New("listener closed")
	}
	return conn, nil
}

func (self *listener) Close() error {
	return self.conn.Close()
}

func (self *listener) Addr() net.Addr {
	return self.addr
}

func (self *listener) run() {
	logrus.Infof("started")
	defer logrus.Warnf("exited")

	for {
		if m, peer, err := readMessage(self.conn); err == nil {
			if lConn, found := self.peers[peer.String()]; !found {
				if m.message == Hello {
					go self.handleHello(peer)
				} else {
					logrus.Errorf("dropping message from unknown peer [%s]", peer)
				}
			} else {
				lConn.queue(m)
			}
		} else {
			logrus.Errorf("read error from [%s] (%v)", peer, err)
			return
		}
	}
}

func (self *listener) handleHello(peer *net.UDPAddr) {
	self.lock.Lock()
	defer self.lock.Unlock()

	conn := newListenerConn(self.conn, self.addr, peer)
	self.peers[peer.String()] = conn

	hello, err := newHelloMessage().marshal()
	if err != nil {
		logrus.Errorf("error creating hello message [%s] (%v)", peer, err)
		return
	}
	n, err := self.conn.WriteTo(hello, peer)
	if err != nil {
		logrus.Errorf("error sending hello [%s] (%v)", peer, err)
		return
	}
	if n != len(hello) {
		logrus.Errorf("short hello [%s]", peer)
		return
	}

	self.acceptQueue <- conn
}

type listener struct {
	lock        *sync.Mutex
	conn        *net.UDPConn
	addr        *net.UDPAddr
	peers       map[string]*listenerConn
	acceptQueue chan net.Conn
}
