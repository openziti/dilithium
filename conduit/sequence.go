package conduit

import "sync"

type sequence struct {
	nextValue int32
	lock      *sync.Mutex
}

func newSequence() *sequence {
	return &sequence{lock: new(sync.Mutex)}
}

func (self *sequence) next() int32 {
	self.lock.Lock()
	n := self.nextValue
	self.nextValue++
	self.lock.Unlock()
	return n
}
