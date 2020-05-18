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
	lastAck  int32
	cConn    net.Conn
	cSeq     *util.Sequence
}

func newRxWindow(cConn net.Conn, cSeq *util.Sequence) *rxWindow {
	rxw := &rxWindow{
		lock:     new(sync.Mutex),
		buffer:   new(bytes.Buffer),
		tree:     btree.NewWith(treeCapacity, utils.Int32Comparator),
		accepted: -1,
		lastAck:  -1,
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
				if self.accepted - self.lastAck > 2 {
					self.txAck(self.accepted, nil)
				}
				next++

				if n, err := self.buffer.Write(m.(*pb.WireMessage).DataPayload.Data); err == nil {
					if n != len(m.(*pb.WireMessage).DataPayload.Data) {
						return errors.New("short buffer write")
					}
					logrus.Infof("[%d] <- {#%d}[%d] <-", self.buffer.Len(), key, n)
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

func (self *rxWindow) eow(highWater int32) {
	self.lock.Lock()
	defer self.lock.Unlock()

	if self.accepted == highWater {
		self.txAck(highWater, nil)
		self.lastAck = highWater

	} else {
		var missing []int32
		for i := self.accepted + 1; i <= highWater; i++ {
			if _, found := self.tree.Get(i); !found {
				missing = append(missing, i)
			}
		}
		self.txAck(self.accepted, missing)
		self.lastAck = self.accepted
	}
}

func (self *rxWindow) txAck(highWater int32, missing []int32) {
	data, err := pb.ToData(pb.NewAck(self.cSeq.Next(), highWater, missing))
	if err != nil {
		logrus.Errorf("[@ %d %v] marshal (%v)", highWater, missing, err)
		return
	}
	n, err := self.cConn.Write(data)
	if err != nil {
		logrus.Errorf("[@ %d %v] write (%v)", highWater, missing, err)
		return
	}
	if n != len(data) {
		logrus.Errorf("[@ %d %v] short write", highWater, missing)
		return
	}
	logrus.Infof("{@ >%d !%v} ->", highWater, missing)
}
