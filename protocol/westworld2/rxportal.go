package westworld2

import (
	"github.com/emirpasic/gods/trees/btree"
	"github.com/emirpasic/gods/utils"
	"github.com/sirupsen/logrus"
	"sync"
)

type rxPortal struct {
	tree        *btree.Tree
	accepted    int32
	rxWmQueue   chan *wireMessage
	rxDataQueue chan *rxRead
	rxDataPool  *sync.Pool
	ackQueue    chan int32
}

type rxRead struct {
	buf []byte
	sz  int
}

func newRxPortal(ackQueue chan int32) *rxPortal {
	pool := new(sync.Pool)
	pool.New = func() interface{} {
		return make([]byte, bufferSz)
	}
	rxp := &rxPortal{
		tree:        btree.NewWith(treeSize, utils.Int32Comparator),
		accepted:    -1,
		rxWmQueue:   make(chan *wireMessage),
		rxDataQueue: make(chan *rxRead, treeSize),
		rxDataPool:  pool,
		ackQueue:    ackQueue,
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
				wm.buffer.unref()
				logrus.Warnf("~ <- {#%d} <-", wm.seq)
			}
			self.ackQueue <- wm.seq

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
