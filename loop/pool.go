package loop

import "sync"

type Pool struct {
	store *sync.Pool
	sz    uint32
}

func NewPool(sz uint32) *Pool {
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
	return newBuffer(self)
}