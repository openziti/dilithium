package westworld2

import (
	"bytes"
	"github.com/emirpasic/gods/trees/btree"
	"github.com/emirpasic/gods/utils"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"sync"
)

type rxPortal struct {
	tree        *btree.Tree
	accepted    int32
	rxWmQueue   chan *wireMessage
	rxBuffer    *bytes.Buffer
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
		rxWmQueue:   make(chan *wireMessage, rxWmQueueSize),
		rxBuffer:    new(bytes.Buffer),
		rxDataQueue: make(chan *rxRead, rxDataQueueSize),
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
		if err := self.rxDataOut(); err != nil {
			logrus.Errorf("rxDataOut (%v)", err)
			return
		}
		if err := self.rxDataIn(); err != nil {
			logrus.Errorf("rxDataIn (%v)", err)
			return
		}
	}
}

func (self *rxPortal) rxDataOut() error {
	if self.rxBuffer.Len() > 0 {
		buf := self.rxDataPool.Get().([]byte)
		n, err := self.rxBuffer.Read(buf)
		if err != nil {
			self.rxDataPool.Put(buf)
			return errors.Errorf("error reading rxBuffer (%v)", err)
		}
		self.rxDataQueue <- &rxRead{buf, n}
	}
	return nil
}

func (self *rxPortal) rxDataIn() error {
	select {
	case wm, ok := <-self.rxWmQueue:
		if !ok {
			return errors.New("rxWmQueue closed")
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
					if n, err := self.rxBuffer.Write(wm.(*wireMessage).data); err == nil {
						if n != len(wm.(*wireMessage).data) {
							return errors.New("short buffer write")
						}

						self.tree.Remove(key)
						self.accepted = next
						next++
					}
				}
			}
		}

	default:
		// no wire messages to receive
	}
	return nil
}
