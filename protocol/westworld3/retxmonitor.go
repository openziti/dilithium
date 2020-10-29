package westworld3

import (
	"github.com/sirupsen/logrus"
	"net"
	"sync"
	"time"
)

type retxMonitor struct {
	profile  *Profile
	ms       int
	conn     *net.UDPConn
	peer     *net.UDPAddr
	waitlist waitlist
	lock     *sync.Mutex
	ready    *sync.Cond
	closed   bool
}

func newRetxMonitor(profile *Profile, conn *net.UDPConn, peer *net.UDPAddr, lock *sync.Mutex) *retxMonitor {
	return &retxMonitor{
		profile:  profile,
		ms:       profile.RetxStartMs,
		conn:     conn,
		peer:     peer,
		waitlist: newArrayWaitlist(),
		lock:     lock,
		ready:    sync.NewCond(lock),
	}
}

func (self *retxMonitor) setMs(ms int) {
	self.ms = ms
}

func (self *retxMonitor) run() {
	logrus.Info("started")
	defer logrus.Warn("exited")

	for {
		var headline time.Time
		var timeout time.Duration

		self.lock.Lock()
		{
			if self.closed {
				return
			}

			for self.waitlist.Size() < 1 {
				self.ready.Wait()
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

						if err := writeWireMessage(wm, self.conn, self.peer); err != nil {
							logrus.Errorf("retx (%v)", err)
						}

						t = time.Now().Add(time.Duration(int(float64(self.ms)*self.profile.RetxScale)+self.profile.RetxAddMs) * time.Millisecond)
						self.waitlist.Add(wm, t)

					} else {
						break
					}
				}
			}
		}
		self.lock.Unlock()
	}
}
