package blaster

import (
	"bytes"
	"github.com/emirpasic/gods/trees/btree"
	"github.com/emirpasic/gods/utils"
	"github.com/michaelquigley/dilithium/protocol/blaster/pb"
	"github.com/michaelquigley/dilithium/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"net"
	"sync"
)

type rxWindow struct {
	lock     *sync.Mutex
	buffer   *bytes.Buffer
	tree     *btree.Tree
	accepted int32
	cConn    net.Conn
	cSeq     *util.Sequence
}

func newRxWindow(cConn net.Conn, cSeq *util.Sequence) *rxWindow {
	rxw := &rxWindow{
		lock:     new(sync.Mutex),
		buffer:   new(bytes.Buffer),
		tree:     btree.NewWith(treeCapacity, utils.Int32Comparator),
		accepted: -1,
		cConn:    cConn,
		cSeq:     cSeq,
	}
	return rxw
}

func (self *rxWindow) rx(wm *pb.WireMessage) error {
	self.lock.Lock()
	defer self.lock.Unlock()

	if wm.Sequence > self.accepted {
		self.tree.Put(wm.Sequence, wm)
	} else {
		logrus.Warnf("dropping already accepted [#%d]", wm.Sequence)
	}

	if self.tree.Size() > 0 {
		next := self.accepted + 1
		for _, key := range self.tree.Keys() {
			if key.(int32) == next {
				m, _ := self.tree.Get(key)
				self.tree.Remove(key)
				self.accepted = next
				next++

				if n, err := self.buffer.Write(m.(*pb.WireMessage).DataPayload.Data); err == nil {
					if n != len(m.(*pb.WireMessage).DataPayload.Data) {
						return errors.New("short buffer write")
					}
					logrus.Infof("(%d) <- [+ #%d](%d) <-", self.buffer.Len(), key, n)
				}
			} else {
				break
			}
		}
	}

	return nil
}

func (self *rxWindow) eow() {
}
