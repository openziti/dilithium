package westworld

import (
	"github.com/emirpasic/gods/trees/btree"
	"github.com/michaelquigley/dilithium/protocol/westworld/wb"
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
	pool        *wb.BufferPool
}

func Listen(addr *net.UDPAddr) (net.Listener, error) {
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, errors.Wrap(err, "listen")
	}
	if err := conn.SetReadBuffer(rxtxBufferSize); err != nil {
		return nil, errors.Wrap(err, "rx buffer")
	}
	if err := conn.SetWriteBuffer(rxtxBufferSize); err != nil {
		return nil, errors.Wrap(err, "tx buffer")
	}
	l := &listener{
		lock:        new(sync.Mutex),
		conn:        conn,
		addr:        addr,
		peers:       btree.NewWith(16*1024, addrComparator),
		acceptQueue: make(chan net.Conn, 1024),
		pool:        wb.NewBufferPool("listener"),
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
		if wm, peer, err := wb.ReadWireMessage(self.conn, self.pool); err == nil {
			wm.Ref()

			conn, found := self.peers.Get(peer)
			if found {
				conn.(*listenerConn).queue(wm)

			} else {
				wm.Touch()
				if wm.Type == wb.HELLO {
					go self.hello(wm, peer)

				} else {
					wm.Unref()
					logrus.Errorf("unknown peer [%s]", peer)
				}
			}
		} else {
			logrus.Errorf("read error from peer [%s] (%v)", peer, err)
		}
	}
}

func (self *listener) hello(hello *wb.WireMessage, peer *net.UDPAddr) {
	conn := newListenerConn(self.conn, peer)

	self.lock.Lock()
	self.peers.Put(peer, conn)
	self.lock.Unlock()

	if err := conn.hello(hello); err != nil {
		logrus.Errorf("hello sequence failed for peer [%s] (%v)", peer, err)
		return
	}

	self.acceptQueue <- conn
	logrus.Infof("accepted connection from [%s]", peer)
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
