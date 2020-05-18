package util

import "sync"

type Sequence struct {
	nextValue int32
	lock      *sync.Mutex
}

func NewSequence(nextValue int32) *Sequence {
	return &Sequence{nextValue: nextValue, lock: new(sync.Mutex)}
}

func (self *Sequence) ResetTo(nextValue int32) {
	self.lock.Lock()
	self.nextValue = nextValue
	self.lock.Unlock()
}

func (self *Sequence) Next() int32 {
	self.lock.Lock()
	n := self.nextValue
	self.nextValue++
	self.lock.Unlock()
	return n
}
