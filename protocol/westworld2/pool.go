package westworld2

import (
	"sync"
)

type pool struct {
	name  string
	store *sync.Pool
	ins   Instrument
}

func newPool(name string, ins Instrument) *pool {
	p := &pool{
		name:  name,
		store: new(sync.Pool),
		ins:   ins,
	}
	p.store.New = p.allocate
	return p
}

func (self *pool) get() *buffer {
	return self.store.Get().(*buffer)
}

func (self *pool) put(buffer *buffer) {
	self.store.Put(buffer)
}

func (self *pool) allocate() interface{} {
	if self.ins != nil {
		self.ins.allocate(self.name)
	}
	return newBuffer(self)
}
