package loop

import "sync"

type pool struct {
	store *sync.Pool
	sz    uint32
}

func newPool(sz uint32) *pool {
	p := &pool{
		store: new(sync.Pool),
		sz:    sz,
	}
	return p
}

func (self *pool) get() *buffer {
	buf := self.store.Get().(*buffer)
	buf.ref()
	return buf
}

func (self *pool) put(buffer *buffer) {
	self.store.Put(buffer)
}

func (self *pool) allocate() interface{} {
	return newBuffer(self)
}