package conduit

import (
	"github.com/emirpasic/gods/trees/btree"
	"github.com/emirpasic/gods/utils"
	"github.com/sirupsen/logrus"
	"math"
	"net"
	"sync"
)

type rxWindow struct {
	buffer     []byte
	tree       *btree.Tree
	lock       *sync.Mutex
	acceptHigh int32
	bufferHigh int32
	conn       *net.UDPConn
	peer       *net.UDPAddr
}

func newRxWindow(conn *net.UDPConn, peer *net.UDPAddr) *rxWindow {
	rxw := &rxWindow{
		tree:       btree.NewWith(10240, utils.Int32Comparator),
		lock:       new(sync.Mutex),
		acceptHigh: -1,
		conn:       conn,
		peer:       peer,
	}
	return rxw
}

func (self *rxWindow) rx(m *message) {
	self.lock.Lock()
	defer self.lock.Unlock()

	if m.sequence > self.acceptHigh {
		self.tree.Put(m.sequence, m)
	} else {
		logrus.Warnf("dropping already received [#%d]", m.sequence)
	}
	self.ack(m)

	if self.tree.Size() > 0 {
		for _, key := range self.tree.Keys() {
			if key.(int32) == self.bufferHigh {
				m, _ := self.tree.Get(key)
				self.tree.Remove(key)
				self.acceptHigh = key.(int32)
				self.bufferHigh++

				self.buffer = append(self.buffer, m.(*message).payload...)

			} else {
				break
			}
		}
	}
}

func (self *rxWindow) ack(m *message) {
	if ack, err := newAckMessage(m.sequence); err == nil {
		if data, err := ack.marshal(); err == nil {
			if n, err := self.conn.WriteTo(data, self.peer); err == nil {
				if n == len(data) {
					logrus.Infof("[@%d] ->", m.sequence)
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

func (self *rxWindow) read(p []byte) (n int, err error) {
	self.lock.Lock()
	defer self.lock.Unlock()

	r := int(math.Min(float64(len(p)), float64(len(self.buffer))))
	if r == 0 {
		return 0, nil
	}

	n = copy(p, self.buffer[:r])
	self.buffer = self.buffer[n:]

	return n, nil
}
