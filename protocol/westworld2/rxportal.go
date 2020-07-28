package westworld2

import (
	"github.com/emirpasic/gods/trees/btree"
	"github.com/emirpasic/gods/utils"
	"github.com/michaelquigley/dilithium/util"
	"github.com/sirupsen/logrus"
	"io"
	"net"
	"sync"
)

type rxPortal struct {
	tree     *btree.Tree
	accepted int32
	closed   bool
	rxs      chan *wireMessage
	reads    chan *rxRead
	readPool *sync.Pool
	ackPool  *pool
	conn     *net.UDPConn
	peer     *net.UDPAddr
	txPortal *txPortal
	seq      *util.Sequence
	config   *Config
}

type rxRead struct {
	buf []byte
	sz  int
	eof bool
}

func newRxPortal(conn *net.UDPConn, peer *net.UDPAddr, txPortal *txPortal, seq *util.Sequence, config *Config) *rxPortal {
	rxp := &rxPortal{
		tree:     btree.NewWith(config.treeLen, utils.Int32Comparator),
		accepted: -1,
		rxs:      make(chan *wireMessage),
		reads:    make(chan *rxRead, config.readsQLen),
		readPool: new(sync.Pool),
		ackPool:  newPool("ackPool", config),
		conn:     conn,
		peer:     peer,
		txPortal: txPortal,
		seq:      seq,
		config:   config,
	}
	rxp.readPool.New = func() interface{} {
		return make([]byte, config.poolBufferSz)
	}
	go rxp.run()
	return rxp
}

func (self *rxPortal) read(p []byte) (int, error) {
	read, ok := <-self.reads
	if !ok {
		return 0, io.EOF
	}
	if !read.eof {
		n := copy(p, read.buf[:read.sz])
		self.readPool.Put(read.buf)
		return n, nil
	} else {
		logrus.Infof("close notified")
		close(self.reads)
		return 0, io.EOF
	}
}

func (self *rxPortal) rx(wm *wireMessage) {
	self.rxs <- wm
}

func (self *rxPortal) setAccepted(accepted int32) {
	self.accepted = accepted
}

func (self *rxPortal) run() {
	logrus.Info("started")
	defer logrus.Warn("exited")

	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("recovered (%v)", r)
		}
	}()

	for {
		wm, ok := <-self.rxs
		if !ok {
			return
		}

		if wm.seq > self.accepted {
			self.tree.Put(wm.seq, wm)
		} else {
			if self.config.i != nil {
				self.config.i.duplicateRx(self.peer, wm)
			}
			wm.buffer.unref()
		}

		ack := newAck(wm.seq, self.ackPool)
		if wm.mf&RTT == 1 {
			ts, err := wm.readRtt()
			if err == nil {
				ack.writeRtt(ts)
			} else {
				logrus.Errorf("error reading rtt (%v)", err)
			}
		}
		if err := writeWireMessage(ack, self.conn, self.peer, self.config.i); err != nil {
			logrus.Errorf("error sending ack (%v)", err)
		}
		ack.buffer.unref()

		if self.tree.Size() > 0 {
			next := self.accepted + 1
			for _, key := range self.tree.Keys() {
				if key.(int32) == next {
					v, _ := self.tree.Get(key)
					wm := v.(*wireMessage)
					buf := self.readPool.Get().([]byte)
					n := copy(buf, wm.data)
					self.reads <- &rxRead{buf, n, false}

					self.tree.Remove(key)
					wm.buffer.unref()
					self.accepted = next
					next++
				}
			}
		}

		if wm.mt == CLOSE && !self.closed {
			self.txPortal.close(self.seq)
			self.reads <- &rxRead{nil, 0, true}
			close(self.rxs)
			self.closed = true
		}
	}
}
