package westworld

import (
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"net"
	"sync"
)

type listener struct {
	lock        *sync.Mutex
	conn        *net.UDPConn
	addr        *net.UDPAddr
	peers       map[string]*listenerConn
	acceptQueue chan net.Conn
	closed      bool
}

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
	self.closed = true // we just want to stop listening
	return nil
}

func (self *listener) Addr() net.Addr {
	return self.addr
}

func (self *listener) run() {
	logrus.Infof("started")
	defer logrus.Warnf("exited")

	for {
	}
}
