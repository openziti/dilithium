package westworld3

import (
	"github.com/pkg/errors"
	"sync"
	"time"
)

type retxMonitor struct {
	waitlist *waitlist
	ready    *sync.Cond
}

type waitlist interface {
	Add(*wireMessage, time.Time)
	Remove(*wireMessage)
	Next() (*wireMessage, time.Time, bool)
}

type retxSubject struct {
	deadline time.Time
	wm       *wireMessage
}

type arrayWaitlist struct {
	waitlist []*retxSubject
}

func (self *arrayWaitlist) Add(wm *wireMessage, t time.Time) error {
	if len(self.waitlist) > 0 {
		if t.Before(self.waitlist[len(self.waitlist)-1].deadline) {
			return errors.New("violated ordering constraint")
		}
	}
	self.waitlist = append(self.waitlist, &retxSubject{t, wm})
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
