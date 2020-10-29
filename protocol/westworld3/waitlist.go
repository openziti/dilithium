package westworld3

import (
	"time"
)

type waitlist interface {
	Add(*wireMessage, time.Time)
	Remove(*wireMessage)
	Size() int
	Peek() (*wireMessage, time.Time)
	Next() (*wireMessage, time.Time)
}

type arrayWaitlist struct {
	waitlist []*waitlistSubject
}

type waitlistSubject struct {
	deadline time.Time
	wm       *wireMessage
}

func newArrayWaitlist() waitlist {
	return &arrayWaitlist{}
}

func (self *arrayWaitlist) Add(wm *wireMessage, t time.Time) {
	self.waitlist = append(self.waitlist, &waitlistSubject{t, wm})
}

func (self *arrayWaitlist) Remove(wm *wireMessage) {
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

func (self *arrayWaitlist) Peek() (*wireMessage, time.Time) {
	if len(self.waitlist) < 1 {
		return nil, time.Time{}
	}
	return self.waitlist[0].wm, self.waitlist[0].deadline
}

func (self *arrayWaitlist) Next() (*wireMessage, time.Time) {
	if len(self.waitlist) < 1 {
		return nil, time.Time{}
	}
	next := self.waitlist[0]
	self.waitlist = self.waitlist[1:]
	return next.wm, next.deadline
}