package util

import "sync"

type Sequence struct {
	nextValue int32
	lock      *sync.Mutex
}

func NewSequence() *Sequence {
	return &Sequence{lock: new(sync.Mutex)}
}

func (self *Sequence) Next() int32 {
	self.lock.Lock()
	n := self.nextValue
	self.nextValue++
	self.lock.Unlock()
	return n
}
