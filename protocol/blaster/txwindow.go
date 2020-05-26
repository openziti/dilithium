package blaster

import (
	"github.com/emirpasic/gods/trees/btree"
	"github.com/emirpasic/gods/utils"
	"github.com/michaelquigley/dilithium/protocol/blaster/pb"
	"github.com/michaelquigley/dilithium/util"
	"github.com/sirupsen/logrus"
	"net"
	"sync"
	"time"
)

const startingWindowCapacity = 4
const treeCapacity = 10240
const eowRequestQueueSz = 64
const minEowPeriod = 200

type txWindow struct {
	lock     *sync.Mutex
	tree     *btree.Tree
	capacity int
	ready    *sync.Cond
	lastAck  time.Time
	eowReq   chan int32
	cConn    net.Conn
	cSeq     *util.Sequence
	dConn    *net.UDPConn
	dPeer    *net.UDPAddr
}

func newTxWindow(cConn net.Conn, cSeq *util.Sequence, dConn *net.UDPConn, dPeer *net.UDPAddr) *txWindow {
	txw := &txWindow{
		lock:     new(sync.Mutex),
		tree:     btree.NewWith(treeCapacity, utils.Int32Comparator),
		capacity: startingWindowCapacity,
		eowReq:   make(chan int32, eowRequestQueueSz),
		cConn:    cConn,
		cSeq:     cSeq,
		dConn:    dConn,
		dPeer:    dPeer,
	}
	txw.ready = sync.NewCond(txw.lock)
	go txw.monitor()
	return txw
}

func (self *txWindow) tx(wm *pb.WireMessage) {
	self.lock.Lock()
	defer self.lock.Unlock()

	for self.capacity < 1 {
		self.eowReq <- self.tree.RightKey().(int32) // request EOW at high water
		self.ready.Wait()
	}

	self.tree.Put(wm.Sequence, wm)
	self.capacity--
}

func (self *txWindow) ack(highWater int32, missing []int32) {
	logrus.Infof("{@ ^%d %v} <-", highWater, missing)

	if len(missing) == 0 {
		logrus.Infof("complete window, pre-ack capacity [%d]", self.capacity)
		if self.tree.Size() > 0 {
			for _, seq := range self.tree.Keys() {
				if seq.(int32) <= highWater {
					self.tree.Remove(seq)
					self.capacity++
				}
			}
			self.ready.Signal()
		}
		logrus.Infof("post-ack capacity [%d]", self.capacity)

	} else {
		for _, m := range missing {
			wm, found := self.tree.Get(m)
			if !found {
				logrus.Errorf("{!#%d} (missing)", m)
				return
			}
			if data, err := pb.ToData(wm.(*pb.WireMessage)); err == nil {
				if n, err := self.dConn.WriteToUDP(data, self.dPeer); err == nil {
					if n == len(data) {
						logrus.Infof("{!#%d} ->", m)
					} else {
						logrus.Errorf("{!#%d} (short write)", m)
					}
				} else {
					logrus.Errorf("{!#%d} (%v)", m, err)
				}
			} else {
				logrus.Errorf("{!#%d} (%v)", m, err)
			}
		}
	}
}

func (self *txWindow) monitor() {
	logrus.Infof("started")
	defer logrus.Warnf("exited")

	for {
		select {
		case _, ok := <-self.eowReq:
			if ok {
				self.txEow()
			}

		case <-time.After(minEowPeriod * time.Millisecond):
			self.txEow()
		}
	}
}

func (self *txWindow) txEow() {
	self.lock.Lock()
	defer self.lock.Unlock()

	if self.tree.Size() > 0 {
		highWater := self.tree.RightKey().(int32)
		data, err := pb.ToData(pb.NewEow(self.cSeq.Next(), highWater))
		if err != nil {
			logrus.Errorf("[^%d] marshal (%v)", highWater, err)
			return
		}
		n, err := self.cConn.Write(data)
		if err != nil {
			logrus.Errorf("[^%d] write (%v)", highWater, err)
			return
		}
		if n != len(data) {
			logrus.Errorf("[^%d] short write", highWater)
			return
		}
		logrus.Infof("[^%d] ->", highWater)
	}
}
