package loop

import (
	"sync"
	"sync/atomic"
)

type Pool struct {
	store       *sync.Pool
	sz          int64
	Allocations int32
}

func NewPool(sz int64) *Pool {
	p := &Pool{
		store: new(sync.Pool),
		sz:    sz,
	}
	p.store.New = p.allocate
	return p
}

func (self *Pool) get() *buffer {
	buf := self.store.Get().(*buffer)
	buf.ref()
	return buf
}

func (self *Pool) put(buffer *buffer) {
	self.store.Put(buffer)
}

func (self *Pool) allocate() interface{} {
	atomic.AddInt32(&self.Allocations, 1)
	return newBuffer(self)
}
