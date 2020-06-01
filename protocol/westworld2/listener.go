package westworld2

import (
	"github.com/emirpasic/gods/trees/btree"
	"github.com/sirupsen/logrus"
	"net"
	"sync"
)

type listener struct {
	lock        *sync.Mutex
	peers       *btree.Tree
	acceptQueue chan net.Conn
	conn        *net.UDPConn
	addr        *net.UDPAddr
	pool        *pool
}

func Listen(addr *net.UDPAddr) (net.Listener, error) {
	return nil, nil
}

func (self *listener) run() {
	logrus.Info("started")
	defer logrus.Warn("exited")

	for {
		if wm, peer, err := readWireMessage(self.conn, self.pool); err == nil {
			conn, found := self.peers.Get(peer)
			if found {
				conn.(*listenerConn).queue(wm)
			} else {
				if wm.mt == HELLO {
					go self.hello(wm, peer)
				} else {
					wm.buffer.unref()
					logrus.Errorf("unknown peer [%s]", peer)
				}
			}
		} else {
			logrus.Errorf("read error, peer [%s] (%v)", peer, err)
		}
	}
}

func (self *listener) hello(hello *wireMessage, peer *net.UDPAddr) {
	conn := newListenerConn(self.conn, peer)

	self.lock.Lock()
	self.peers.Put(peer, conn)
	self.lock.Unlock()

	if err := conn.hello(hello); err != nil {
		logrus.Errorf("connect establishment failed, peer [%s] (%v)", peer, err)
		return
	}

	self.acceptQueue <- conn
	logrus.Infof("accepted connection, peer [%s]", peer)
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
