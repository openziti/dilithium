package westworld2

import (
	"github.com/emirpasic/gods/trees/btree"
	"github.com/emirpasic/gods/utils"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"net"
	"sort"
	"time"
)

type txPortal struct {
	tree     *btree.Tree
	capacity int
	txQueue  chan *wireMessage
	txErrors chan error
	rxAcks   chan int32
	retxList []*retx
	conn     *net.UDPConn
	peer     *net.UDPAddr
	pool     *pool
	ins      Instrument
}

type retx struct {
	deadline time.Time
	wm       *wireMessage
}

func newTxPortal(conn *net.UDPConn, peer *net.UDPAddr, ins Instrument) *txPortal {
	txp := &txPortal{
		tree:     btree.NewWith(treeSize, utils.Int32Comparator),
		capacity: startingWindowCapacity,
		txQueue:  make(chan *wireMessage, startingWindowCapacity),
		txErrors: make(chan error, txErrorsSize),
		rxAcks:   make(chan int32, ackQueueSize),
		conn:     conn,
		peer:     peer,
		pool:     newPool("txPortal", ins),
		ins:      ins,
	}
	go txp.run()
	return txp
}

func (self *txPortal) run() {
	logrus.Info("started")
	defer logrus.Warn("exited")

	for {
		if err := self.doAcks(); err != nil {
			logrus.Errorf("doAcks (%v)", err)
			return
		}
		if err := self.doRetx(); err != nil {
			logrus.Errorf("doRetx (%v)", err)
			return
		}
		if err := self.doTx(); err != nil {
			logrus.Errorf("doTx (%v)", err)
			return
		}
	}
}

func (self *txPortal) doAcks() error {
	for {
		select {
		case rxAck, ok := <-self.rxAcks:
			if !ok {
				return errors.New("rxAcks closed")
			}

			if wm, found := self.tree.Get(rxAck); found {
				self.tree.Remove(rxAck)
				wm.(*wireMessage).buffer.unref()
				self.capacity++
				self.cancelRetx(wm.(*wireMessage))

			} else {
				logrus.Warnf("~ [@%d] <- (capacity: %d)", rxAck, self.capacity) // already acked
				for i, re := range self.retxList {
					logrus.Infof("retxList [%d] = #%d", i, re.wm.seq)
				}
			}

		default:
			// no acks to process
			return nil
		}
	}
}

func (self *txPortal) doRetx() error {
	if len(self.retxList) > 0 {
		re := self.retxList[0]
		if time.Since(re.deadline).Milliseconds() > retxTimeoutMs {
			if self.ins != nil {
				self.ins.wireMessageRetx(self.peer, re.wm)
			}

			if err := writeWireMessage(re.wm, self.conn, self.peer, self.ins); err != nil {
				return errors.Wrap(err, "retx write")
			}
			re.deadline = time.Now().Add(retxTimeoutMs * time.Millisecond)
			if len(self.retxList) > 1 {
				self.retxList = append(self.retxList[1:], self.retxList[0])
				self.sortRetx()
			}
		}
	}
	return nil
}

func (self *txPortal) doTx() error {
	for self.capacity > 0 {
		select {
		case wm, ok := <-self.txQueue:
			if !ok {
				return errors.New("txQueue closed")
			}

			self.tree.Put(wm.seq, wm)
			self.capacity--
			self.addRetx(wm)

			if err := writeWireMessage(wm, self.conn, self.peer, self.ins); err != nil {
				logrus.Errorf("-> {#%d,@%d}[%d] -> (%v)", wm.seq, wm.ack, len(wm.data), err)
				self.txErrors <- errors.Wrap(err, "write")
			}

		default:
			// no wireMessage for tx
			return nil
		}
	}
	return nil
}

func (self *txPortal) addRetx(wm *wireMessage) {
	deadline := time.Now().Add(retxTimeoutMs * time.Millisecond)
	self.retxList = append(self.retxList, &retx{deadline, wm})
	self.sortRetx()
}

func (self *txPortal) cancelRetx(wm *wireMessage) {
	i := -1
	for j, c := range self.retxList {
		if c.wm == wm {
			i = j
			break
		}
	}
	if i > -1 {
		self.retxList = append(self.retxList[:i], self.retxList[i+1:]...)
		self.sortRetx()
	}
}

func (self *txPortal) sortRetx() {
	sort.Slice(self.retxList, func(i, j int) bool {
		return self.retxList[i].deadline.Before(self.retxList[j].deadline)
	})
}
