package westworld2

import (
	"github.com/emirpasic/gods/trees/btree"
	"github.com/pkg/errors"
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
	ins         instrument
}

func Listen(addr *net.UDPAddr) (net.Listener, error) {
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, errors.Wrap(err, "listen")
	}
	if err := conn.SetReadBuffer(bufferSz); err != nil {
		return nil, errors.Wrap(err, "rx buffer")
	}
	if err := conn.SetWriteBuffer(bufferSz); err != nil {
		return nil, errors.Wrap(err, "tx buffer")
	}
	l := &listener{
		lock:        new(sync.Mutex),
		peers:       btree.NewWith(treeSize, addrComparator),
		acceptQueue: make(chan net.Conn, acceptQueueSz),
		conn:        conn,
		addr:        addr,
		pool:        newPool("listener"),
		ins:         &loggerInstrument{},
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
	return nil
}

func (self *listener) Addr() net.Addr {
	return self.addr
}

func (self *listener) run() {
	logrus.Info("started")
	defer logrus.Warn("exited")

	for {
		if wm, peer, err := readWireMessage(self.conn, self.pool, self.ins); err == nil {
			conn, found := self.peers.Get(peer)
			if found {
				conn.(*listenerConn).queue(wm)
			} else {
				if wm.mt == HELLO {
					go self.hello(wm, peer)

				} else {
					wm.buffer.unref()
					if self.ins != nil {
						self.ins.unknownPeer(peer)
					}
				}
			}
		} else {
			if self.ins != nil {
				self.ins.readError(peer, err)
			}
		}
	}
}

func (self *listener) hello(hello *wireMessage, peer *net.UDPAddr) {
	conn := newListenerConn(self.conn, peer, &loggerInstrument{})

	self.lock.Lock()
	self.peers.Put(peer, conn)
	self.lock.Unlock()

	if err := conn.hello(hello); err != nil {
		if self.ins != nil {
			self.ins.connectError(peer, err)
		}
		return
	}

	self.acceptQueue <- conn

	if self.ins != nil {
		self.ins.connected(peer)
	}
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
