package westworld3

import (
	"github.com/openziti/dilithium/util"
	"github.com/sirupsen/logrus"
	"net"
	"sync"
	"time"
)

type retxMonitor struct {
	profile  *Profile
	rttAvg   []uint16
	retxMs   int
	conn     *net.UDPConn
	peer     *net.UDPAddr
	waitlist waitlist
	lock     *sync.Mutex
	ready    *sync.Cond
	closed   bool
	retxF    func()
	ii       InstrumentInstance
}

func newRetxMonitor(profile *Profile, conn *net.UDPConn, peer *net.UDPAddr, lock *sync.Mutex, ii InstrumentInstance) *retxMonitor {
	rm := &retxMonitor{
		profile:  profile,
		retxMs:   profile.RetxStartMs,
		conn:     conn,
		peer:     peer,
		waitlist: newArrayWaitlist(),
		lock:     lock,
		ready:    sync.NewCond(lock),
		ii:       ii,
	}
	go rm.run()
	return rm
}

func (self *retxMonitor) setRetxF(f func()) {
	self.retxF = f
}

func (self *retxMonitor) updateRttMs(rttMs uint16) {
	self.rttAvg = append(self.rttAvg, rttMs)
	if len(self.rttAvg) > self.profile.RttProbeAvg {
		self.rttAvg = self.rttAvg[1:]
	}
	accum := 0
	for _, rttMs := range self.rttAvg {
		accum += int(rttMs)
	}
	accum /= len(self.rttAvg)
	self.retxMs = int(float64(accum)*self.profile.RetxScale) + self.profile.RetxAddMs
	self.ii.NewRetxMs(self.peer, self.retxMs)
}

func (self *retxMonitor) monitor(wm *wireMessage) {
	self.waitlist.Add(wm, self.deadline())
	self.ready.Broadcast()
}

func (self *retxMonitor) cancel(wm *wireMessage) {
	self.waitlist.Remove(wm)
}

func (self *retxMonitor) close() {
	self.closed = true
	self.ready.Broadcast()
}

func (self *retxMonitor) run() {
	logrus.Info("started")
	defer logrus.Warn("exited")

	for {
		var headline time.Time
		var timeout time.Duration

		self.lock.Lock()
		{
			for self.waitlist.Size() < 1 && !self.closed {
				self.ready.Wait()
			}

			if self.closed {
				return
			}

			_, headline = self.waitlist.Peek()
			timeout = time.Until(headline)
		}
		self.lock.Unlock()

		time.Sleep(timeout)

		self.lock.Lock()
		{
			if self.waitlist.Size() > 0 {
				i := 0
				x := self.waitlist.Size()
				for ; i < x; i++ {
					_, t := self.waitlist.Peek()
					delta := t.Sub(headline).Milliseconds()
					if delta <= int64(self.profile.RetxBatchMs) {
						wm, _ := self.waitlist.Next()
						if wm.hasFlag(RTT) {
							util.WriteUint16(wm.buffer.data[dataStart:], uint16(time.Now().UnixNano()/int64(time.Millisecond)))
						}

						if err := writeWireMessage(wm, self.conn, self.peer); err != nil {
							logrus.Errorf("retx (%v)", err)
						} else {
							self.ii.WireMessageRetx(self.peer, wm)
						}
						if self.retxF != nil {
							self.retxF()
						}

						self.waitlist.Add(wm, self.deadline())

					} else {
						break
					}
				}
			}
		}
		self.lock.Unlock()
	}
}

func (self *retxMonitor) deadline() time.Time {
	return time.Now().Add(time.Duration(self.retxMs) * time.Millisecond)
}
