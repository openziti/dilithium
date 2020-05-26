package westworld

import (
	"bytes"
	"github.com/emirpasic/gods/trees/btree"
	"github.com/emirpasic/gods/utils"
	"github.com/michaelquigley/dilithium/protocol/westworld/pb"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"net"
	"sync"
)

type rxWindow struct {
	lock       *sync.Mutex
	buffer     *bytes.Buffer
	tree       *btree.Tree
	accepted   int32
	ackQueue   chan int32
	conn       *net.UDPConn
	peer       *net.UDPAddr
	txWindow   *txWindow
}

func newRxWindow(ackQueue chan int32, conn *net.UDPConn, peer *net.UDPAddr, txWindow *txWindow) *rxWindow {
	rxw := &rxWindow{
		lock:       new(sync.Mutex),
		buffer:     new(bytes.Buffer),
		tree:       btree.NewWith(startingTreeSize, utils.Int32Comparator),
		accepted:   -1,
		ackQueue:   ackQueue,
		conn:       conn,
		peer:       peer,
		txWindow:   txWindow,
	}
	return rxw
}

func (self *rxWindow) rx(wm *pb.WireMessage) error {
	self.lock.Lock()
	defer self.lock.Unlock()

	if wm.Sequence > self.accepted {
		self.tree.Put(wm.Sequence, wm)
	} else {
		logrus.Warnf("~ <- {#%d} <-", wm.Sequence)
	}
	self.txWindow.ackQueue <- wm.Sequence

	if self.tree.Size() > 0 {
		next := self.accepted + 1
		for _, key := range self.tree.Keys() {
			if key.(int32) == next {
				wm, _ := self.tree.Get(key)
				self.tree.Remove(key)
				self.accepted = next
				next++

				if n, err := self.buffer.Write(wm.(*pb.WireMessage).Data); err == nil {
					if n != len(wm.(*pb.WireMessage).Data) {
						return errors.New("short buffer write")
					}
					//logrus.Infof("[%d] <- {#%d}[%d] <-", self.buffer.Len(), key, n)
				}

			} else {
				break
			}
		}
	}

	return nil
}

func (self *rxWindow) read(p []byte) (n int, err error) {
	self.lock.Lock()
	defer self.lock.Unlock()
	return self.buffer.Read(p)
}