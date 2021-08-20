package dilithium

import "time"

// TxMonitor is responsible for managing in-flight payloads, retransmitting payloads when their timeout expires.
//
type TxMonitor struct {
	alg       TxAlgorithm
	transport Transport
	rttAvg    []uint16
	retxMs    int
}

type waitlist interface {
	Add(*WireMessage, int, time.Time)
	Update(int)
	Remove(*WireMessage)
	Size() int
	Peek() (*WireMessage, time.Time)
	Next() (*WireMessage, time.Time)
}

type arrayWaitlist struct {
	waitlist []*waitlistSubject
}

type waitlistSubject struct {
	deadline time.Time
	retxMs   int
	wm       *WireMessage
}

func newArrayWaitlist() waitlist {
	return &arrayWaitlist{}
}

func (self *arrayWaitlist) Add(wm *WireMessage, retxMs int, t time.Time) {
	self.waitlist = append(self.waitlist, &waitlistSubject{t, retxMs, wm})
}

func (self *arrayWaitlist) Update(retxMs int) {
	for _, waiter := range self.waitlist {
		delta := retxMs - waiter.retxMs
		waiter.deadline.Add(time.Duration(delta) * time.Millisecond)
	}
}

func (self *arrayWaitlist) Remove(wm *WireMessage) {
	i := -1
	for i = 0; i < len(self.waitlist); i++ {
		if self.waitlist[i].wm == wm {
			break
		}
	}
	if i > -1 {
		self.waitlist = append(self.waitlist[:i], self.waitlist[i+1:]...)
	}
}

func (self *arrayWaitlist) Size() int {
	return len(self.waitlist)
}

func (self *arrayWaitlist) Peek() (*WireMessage, time.Time) {
	if len(self.waitlist) < 1 {
		return nil, time.Time{}
	}
	return self.waitlist[0].wm, self.waitlist[0].deadline
}

func (self *arrayWaitlist) Next() (*WireMessage, time.Time) {
	if len(self.waitlist) < 1 {
		return nil, time.Time{}
	}
	next := self.waitlist[0]
	self.waitlist = self.waitlist[1:]
	return next.wm, next.deadline
}