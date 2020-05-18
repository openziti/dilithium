package westworld

import (
	"github.com/emirpasic/gods/trees/btree"
	"github.com/michaelquigley/dilithium/protocol/westworld/pb"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"net"
	"sync"
)

type listener struct {
	lock        *sync.Mutex
	conn        *net.UDPConn
	addr        *net.UDPAddr
	peers       *btree.Tree
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
		peers:       btree.NewWith(16*1024, addrComparator),
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
		if wm, peer, err := pb.ReadWireMessage(self.conn); err == nil {
			conn, found := self.peers.Get(peer)
			if found {
				conn.(*listenerConn).queue(wm)

			} else {
				if wm.Type == pb.MessageType_HELLO {
					go self.hello(peer)

				} else {
					logrus.Errorf("unknown peer [%s]", peer)
				}
			}
		} else {
			logrus.Errorf("read error from peer [%s] (%v)", peer, err)
		}
	}
}

func (self *listener) hello(peer *net.UDPAddr) {
	self.lock.Lock()
	defer self.lock.Unlock()
}

func addrComparator(i, j interface{}) int {
	ai := i.(*net.UDPAddr)
	aj := j.(*net.UDPAddr)
	for l := 0; l < 4; l++ {
		if ai.IP[l] < aj.IP[l] {
			return -1
		}
		if ai.IP[l] > aj.IP[l] {
			return 1
		}
	}
	if ai.Port < aj.Port {
		return -1
	}
	if ai.Port > aj.Port {
		return 1
	}
	return 0
}
