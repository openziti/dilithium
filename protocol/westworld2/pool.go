package westworld2

import (
	"sync"
)

type pool struct {
	name  string
	store *sync.Pool
	ii    InstrumentInstance
}

func newPool(name string, ii InstrumentInstance) *pool {
	p := &pool{
		name:  name,
		store: new(sync.Pool),
		ii:    ii,
	}
	p.store.New = p.allocate
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
	if self.ii != nil {
		self.ii.allocate(self.name)
	}
	return newBuffer(self)
}
