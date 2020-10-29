package westworld3

import (
	"github.com/sirupsen/logrus"
	"net"
	"sync"
	"time"
)

type retxMonitor struct {
	waitlist waitlist
	lock     *sync.Mutex
	ready    *sync.Cond
	conn     *net.UDPConn
	peer     *net.UDPAddr
	ii       InstrumentInstance
	closed   bool
}

func newRetxMonitor(lock *sync.Mutex, conn *net.UDPConn, peer *net.UDPAddr, ii InstrumentInstance) *retxMonitor {
	return &retxMonitor{
		waitlist: newArrayWaitlist(),
		lock:     lock,
		ready:    sync.NewCond(lock),
		ii:       ii,
	}
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
	}
}
