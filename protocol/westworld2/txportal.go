package westworld2

import (
	"github.com/emirpasic/gods/trees/btree"
	"github.com/emirpasic/gods/utils"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"net"
	"time"
)

type txPortal struct {
	tree     *btree.Tree
	capacity int
	txQueue  chan *wireMessage
	txErrors chan error
	rxAcks   chan int32
	txAcks   chan int32
	retxList []*retx
	conn     *net.UDPConn
	peer     *net.UDPAddr
	pool     *pool
	ins      instrument
}

type retx struct {
	deadline time.Time
	wm       *wireMessage
}

func newTxPortal(conn *net.UDPConn, peer *net.UDPAddr, ins instrument) *txPortal {
	txp := &txPortal{
		tree:     btree.NewWith(treeSize, utils.Int32Comparator),
		capacity: startingWindowCapacity,
		txQueue:  make(chan *wireMessage, startingWindowCapacity),
		txErrors: make(chan error, txErrorsSize),
		rxAcks:   make(chan int32, ackQueueSize),
		txAcks:   make(chan int32, ackQueueSize),
		conn:     conn,
		peer:     peer,
		pool:     newPool("txPortal"),
		ins:      ins,
	}
	go txp.run()
	return txp
}

func (self *txPortal) run() {
	logrus.Info("started")
	defer logrus.Warn("exited")

	for {
		if err := self.rxtxAcks(); err != nil {
			logrus.Errorf("rxtxAcks (%v)", err)
			return
		}
		if err := self.retxData(); err != nil {
			logrus.Errorf("retxData (%v)", err)
			return
		}
		if err := self.txData(); err != nil {
			logrus.Errorf("txData (%v)", err)
			return
		}
	}
}

func (self *txPortal) rxtxAcks() error {
	select {
	case txAck, ok := <-self.txAcks:
		if !ok {
			return errors.New("txAcks closed")
		}

		wm := newAck(txAck, self.pool)
		if err := writeWireMessage(wm, self.conn, self.peer); err != nil {
			logrus.Errorf("{@%d} -> (%v)", txAck, err)
		}
		wm.buffer.unref()

	case rxAck, ok := <-self.rxAcks:
		if !ok {
			return errors.New("rxAcks closed")
		}

		if wm, found := self.tree.Get(rxAck); found {
			self.tree.Remove(rxAck)
			wm.(*wireMessage).buffer.unref()
			self.capacity++
			self.cancelReTx(wm.(*wireMessage))

		} else {
			logrus.Warnf("~ [@%d] <-", rxAck) // already acked
		}

	default:
		// no acks to process
	}
	return nil
}

func (self *txPortal) retxData() error {
	if len(self.retxList) > 0 {
		re := self.retxList[0]
		if time.Since(re.deadline).Milliseconds() > retxTimeoutMs {
			logrus.Infof("retx [#%d]", re.wm.seq)
			if err := writeWireMessage(re.wm, self.conn, self.peer); err != nil {
				return errors.Wrap(err, "retx write")
			}
			re.deadline = time.Now().Add(retxTimeoutMs * time.Millisecond)
			if len(self.retxList) > 1 {
				self.retxList = append(self.retxList[:1], self.retxList[0])
			}
		}
	}
	return nil
}

func (self *txPortal) txData() error {
	if self.capacity > 0 {
		select {
		case wm, ok := <-self.txQueue:
			if !ok {
				return errors.New("txQueue closed")
			}

			self.tree.Put(wm.seq, wm)
			self.capacity--
			self.addReTx(wm)

			select {
			case ack, ok := <-self.txAcks:
				if !ok {
					return errors.New("txAcks closed")
				}
				wm.rewriteAck(ack)
			default:
				// no ack for slipstream
			}

			if err := writeWireMessage(wm, self.conn, self.peer); err != nil {
				logrus.Errorf("-> {#%d,@%d}[%d] -> (%v)", wm.seq, wm.ack, len(wm.data), err)
				self.txErrors <- errors.Wrap(err, "write")
			}
			if self.ins != nil {
				self.ins.wireMessageTx(wm)
			}

		default:
			// no wireMessage for tx
		}
	}
	return nil
}

func (self *txPortal) addReTx(wm *wireMessage) {
	deadline := time.Now().Add(retxTimeoutMs * time.Millisecond)
	self.retxList = append(self.retxList, &retx{deadline, wm})
}

func (self *txPortal) cancelReTx(wm *wireMessage) {
	i := -1
	for j, c := range self.retxList {
		if c.wm == wm {
			i = j
			break
		}
	}
	if i > -1 {
		self.retxList = append(self.retxList[:i], self.retxList[i+1:]...)
	}
}
