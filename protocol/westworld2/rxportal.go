package westworld2

import (
	"github.com/emirpasic/gods/trees/btree"
	"github.com/emirpasic/gods/utils"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"net"
	"sync"
)

type rxPortal struct {
	tree     *btree.Tree
	accepted int32
	rxs      chan *wireMessage
	reads    chan *rxRead
	readPool *sync.Pool
	ackPool  *pool
	conn     *net.UDPConn
	peer     *net.UDPAddr
	ins      Instrument
}

type rxRead struct {
	buf []byte
	sz  int
}

func newRxPortal2(conn *net.UDPConn, peer *net.UDPAddr, ins Instrument) *rxPortal {
	rxp := &rxPortal{
		tree:     btree.NewWith(treeSize, utils.Int32Comparator),
		accepted: -1,
		rxs:      make(chan *wireMessage),
		reads:    make(chan *rxRead, rxQueueSize),
		readPool: new(sync.Pool),
		ackPool:  newPool("ackPool", ins),
		conn:     conn,
		peer:     peer,
		ins:      ins,
	}
	rxp.readPool.New = func() interface{} {
		return make([]byte, bufferSz)
	}
	go rxp.run()
	return rxp
}

func (self *rxPortal) read(p []byte) (int, error) {
	read, ok := <-self.reads
	if !ok {
		return 0, errors.New("closed")
	}
	n := copy(p, read.buf[:read.sz])
	self.readPool.Put(read.buf)
	return n, nil
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

	for {
		wm, ok := <-self.rxs
		if !ok {
			return
		}

		if wm.seq > self.accepted {
			self.tree.Put(wm.seq, wm)
		} else {
			if self.ins != nil {
				self.ins.duplicateRx(self.peer, wm)
			}
		}

		ack := newAck(wm.seq, self.ackPool)
		if err := writeWireMessage(ack, self.conn, self.peer, self.ins); err != nil {
			logrus.Errorf("error sending ack (%v)", err)
		}
		ack.buffer.unref()

		if self.tree.Size() > 0 {
			next := self.accepted + 1
			for _, key := range self.tree.Keys() {
				if key.(int32) == next {
					wm, _ := self.tree.Get(key)
					buf := self.readPool.Get().([]byte)
					n := copy(buf, wm.(*wireMessage).data)
					self.reads <- &rxRead{buf, n}

					self.tree.Remove(key)
					wm.(*wireMessage).buffer.unref()
					self.accepted = next
					next++
				}
			}
		}
	}
}
