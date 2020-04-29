package conduit

import (
	"github.com/emirpasic/gods/trees/btree"
	"github.com/emirpasic/gods/utils"
	"github.com/sirupsen/logrus"
	"net"
	"sync"
)

type rxWindow struct {
	tree *btree.Tree
	lock *sync.Mutex
	high int32
	conn *net.UDPConn
	peer *net.UDPAddr
}

func newRxWindow(conn *net.UDPConn, peer *net.UDPAddr) *rxWindow {
	rxw := &rxWindow{
		tree: btree.NewWith(10240, utils.Int32Comparator),
		lock: new(sync.Mutex),
		high: -1,
		conn: conn,
		peer: peer,
	}
	return rxw
}

func (self *rxWindow) rx(m *message) []*message {
	self.lock.Lock()
	defer self.lock.Unlock()

	if m.sequence > self.high {
		self.tree.Put(m.sequence, m)
	} else {
		logrus.Warnf("dropping already received [#%d]", m.sequence)
	}
	self.ack(m)

	var output []*message
	if self.tree.Size() > 0 {
		next := self.tree.LeftKey().(int32)
		for _, key := range self.tree.Keys() {
			if key.(int32) == next {
				m, _ := self.tree.Get(key)
				self.tree.Remove(key)
				output = append(output, m.(*message))
				self.high = key.(int32)
				next++
			} else {
				break
			}
		}
	}
	return output
}

func (self *rxWindow) ack(m *message) {
	if ack, err := newAckMessage(m.sequence); err == nil {
		if data, err := ack.marshal(); err == nil {
			if n, err := self.conn.WriteTo(data, self.peer); err == nil {
				if n == len(data) {
					logrus.Infof("[@ %d] ->", m.sequence)
				} else {
					logrus.Errorf("ack, short write")
				}
			} else {
				logrus.Errorf("ack, write error (%v)", err)
			}
		} else {
			logrus.Errorf("ack, marshal error (%v)", err)
		}
	} else {
		logrus.Errorf("ack, message error (%v)", err)
	}
}