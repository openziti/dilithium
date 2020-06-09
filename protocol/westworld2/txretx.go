package westworld2

import (
	"github.com/sirupsen/logrus"
	"net"
	"time"
)

const monitorInSz = 1024
const cancelInSz = 1024

type txRetx struct {
	monitorIn chan *wireMessage
	cancelIn  chan int32
	doneIn    chan *txRetxMonitor
	queue     []*txRetxMonitor
	conn      *net.UDPConn
	peer      *net.UDPAddr
	ins       Instrument
}

type txRetxMonitor struct {
	deadline  time.Time
	cancelled bool
	wm        *wireMessage
}

func newTxRetx(conn *net.UDPConn, peer *net.UDPAddr) *txRetx {
	tr := &txRetx{
		monitorIn: make(chan *wireMessage, monitorInSz),
		cancelIn:  make(chan int32, cancelInSz),
		doneIn:    make(chan *txRetxMonitor, cancelInSz),
		conn:      conn,
		peer:      peer,
	}
	go tr.run()
	return tr
}

func (self *txRetx) run() {
	for {
		select {
		case ack, ok := <-self.cancelIn:
			if !ok {
				return
			}

			done := false
			for !done {
				i := -1
			search:
				for j, c := range self.queue {
					if c.wm.seq == ack {
						i = j
						break search
					}
				}

				if i > -1 {
					self.queue[i].cancelled = true
					self.queue = append(self.queue[:i], self.queue[i+1:]...)
				} else {
					done = true
				}
			}

		case rem, ok := <-self.doneIn:
			if !ok {
				return
			}

			if !rem.cancelled {
				rem.deadline = time.Now().Add(retxTimeoutMs * time.Millisecond)
				go self.retxer(rem)
			} else {
				//rem.wm.buffer.unref()
			}

		case wm, ok := <-self.monitorIn:
			if !ok {
				return
			}
			wm.buffer.ref()
			rem := &txRetxMonitor{time.Now().Add(retxTimeoutMs * time.Millisecond), false, wm}
			self.queue = append(self.queue, rem)
			go self.retxer(rem)
		}
	}
}

func (self *txRetx) retxer(rem *txRetxMonitor) {
	time.Sleep(time.Until(rem.deadline))
	if !rem.cancelled {
		if err := writeWireMessage(rem.wm, self.conn, self.peer, self.ins); err != nil {
			logrus.Errorf("retx (%v)", err)
		}
	}
	self.doneIn <- rem
}
