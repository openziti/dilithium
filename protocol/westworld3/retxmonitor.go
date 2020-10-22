package westworld3

import (
	"github.com/emirpasic/gods/trees/btree"
	"github.com/emirpasic/gods/utils"
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

type btreeWaitlist struct {
	waitlist *btree.Tree
}

func newBtreeWaitlist() *btreeWaitlist {
	return &btreeWaitlist{waitlist: btree.NewWith(1024, utils.TimeComparator)}
}

func (self *btreeWaitlist) Add(wm *wireMessage, t time.Time) error {
	if v, found := self.waitlist.Get(t); found {
		switch v.(type) {
		case *wireMessage:
			vAsserted := v.(*wireMessage)
			if vAsserted != wm {
				tl := btree.NewWith(128, wireMessageComparator)
				tl.Put(vAsserted, nil)
				tl.Put(wm, nil)
				self.waitlist.Put(t, tl)
			}

		case *btree.Tree:
			vAsserted := v.(*btree.Tree)
			vAsserted.Put(wm, nil)
		}
	}
	return nil
}

func wireMessageComparator(a interface{}, b interface{}) int {
	aAsserted := a.(*wireMessage)
	bAsserted := b.(*wireMessage)
	if aAsserted.seq > bAsserted.seq {
		return 1
	} else if aAsserted.seq < bAsserted.seq {
		return -1
	} else {
		return 0
	}
}