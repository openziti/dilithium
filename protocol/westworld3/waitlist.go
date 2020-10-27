package westworld3

import (
	"time"
)

type waitlistSubject struct {
	deadline time.Time
	wm       *wireMessage
}

type arrayWaitlist struct {
	waitlist []*waitlistSubject
}

func (self *arrayWaitlist) Add(wm *wireMessage, t time.Time) error {
	self.waitlist = append(self.waitlist, &waitlistSubject{t, wm})
	return nil
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

func (self *arrayWaitlist) Next() (*wireMessage, time.Time) {
	if len(self.waitlist) < 1 {
		return nil, time.Time{}
	}
	next := self.waitlist[0]
	self.waitlist = self.waitlist[1:]
	return next.wm, next.deadline
}