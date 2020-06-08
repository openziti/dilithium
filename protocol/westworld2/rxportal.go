package westworld2

import (
	"github.com/emirpasic/gods/trees/btree"
	"github.com/emirpasic/gods/utils"
	"github.com/sirupsen/logrus"
	"net"
	"sync"
)

type rxPortal struct {
	tree        *btree.Tree
	accepted    int32
	rxWmQueue   chan *wireMessage
	rxDataQueue chan *rxRead
	rxDataPool  *sync.Pool
	txPool      *pool
	conn        *net.UDPConn
	peer        *net.UDPAddr
	ins         Instrument
}

type rxRead struct {
	buf []byte
	sz  int
}

func newRxPortal(conn *net.UDPConn, peer *net.UDPAddr, ins Instrument) *rxPortal {
	pool := new(sync.Pool)
	pool.New = func() interface{} {
		return make([]byte, bufferSz)
	}
	rxp := &rxPortal{
		tree:        btree.NewWith(treeSize, utils.Int32Comparator),
		accepted:    -1,
		rxWmQueue:   make(chan *wireMessage),
		rxDataQueue: make(chan *rxRead, rxDataQueueSize),
		rxDataPool:  pool,
		conn:        conn,
		peer:        peer,
		txPool:      newPool("txPool", ins),
		ins:         ins,
	}
	go rxp.run()
	return rxp
}

func (self *rxPortal) run() {
	logrus.Info("started")
	defer logrus.Warn("exited")

	for {
		select {
		case wm, ok := <-self.rxWmQueue:
			if !ok {
				return
			}

			if wm.seq > self.accepted {
				self.tree.Put(wm.seq, wm)
			} else {
				logrus.Warnf("~ <- {#%d} <-", wm.seq)
			}

			ack := newAck(wm.seq, self.txPool)
			if err := writeWireMessage(ack, self.conn, self.peer, self.ins); err != nil {
				logrus.Errorf("error sending ack (%v)", err)
			}
			ack.buffer.unref()

			if self.tree.Size() > 0 {
				next := self.accepted + 1
				for _, key := range self.tree.Keys() {
					if key.(int32) == next {
						wm, _ := self.tree.Get(key)
						buf := self.rxDataPool.Get().([]byte)
						n := copy(buf, wm.(*wireMessage).data)
						self.rxDataQueue <- &rxRead{buf, n}

						self.tree.Remove(key)
						wm.(*wireMessage).buffer.unref()
						self.accepted = next
						next++
					}
				}
			}

		default:
			// no wire messages to receive
		}
	}
}
