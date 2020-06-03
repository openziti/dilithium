package conduit

import (
	"bytes"
	"github.com/emirpasic/gods/trees/btree"
	"github.com/emirpasic/gods/utils"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"net"
	"sync"
)

type rxWindow struct {
	buffer   *bytes.Buffer
	tree     *btree.Tree
	lock     *sync.Mutex
	accepted int32
	conn     *net.UDPConn
	peer     *net.UDPAddr
}

func newRxWindow(conn *net.UDPConn, peer *net.UDPAddr) *rxWindow {
	rxw := &rxWindow{
		buffer:   new(bytes.Buffer),
		tree:     btree.NewWith(10240, utils.Int32Comparator),
		lock:     new(sync.Mutex),
		accepted: -1,
		conn:     conn,
		peer:     peer,
	}
	return rxw
}

func (self *rxWindow) rx(m *message) error {
	self.lock.Lock()
	defer self.lock.Unlock()

	if m.sequence > self.accepted {
		self.tree.Put(m.sequence, m)
	} else {
		logrus.Warnf("dropping already accepted [#%d]", m.sequence)
	}
	self.ack(m)

	if self.tree.Size() > 0 {
		next := self.accepted + 1
		for _, key := range self.tree.Keys() {
			if key.(int32) == next {
				m, _ := self.tree.Get(key)
				self.tree.Remove(key)
				self.accepted = next
				next++

				if n, err := self.buffer.Write(m.(*message).payload); err == nil {
					if n != len(m.(*message).payload) {
						return errors.New("short buffer writes")
					}
					//logrus.Infof("(%d) <- [+ #%d](%d) <-", self.buffer.Len(), key, n)
				}

			} else {
				break
			}
		}
	}

	return nil
}

func (self *rxWindow) ack(m *message) {
	if ack, err := newAckMessage(m.sequence); err == nil {
		if data, err := ack.marshal(); err == nil {
			if n, err := self.conn.WriteTo(data, self.peer); err == nil {
				if n == len(data) {
					//logrus.Infof("[@%d] ->", m.sequence)
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
	return self.buffer.Read(p)
}
