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
	buf := self.store.Get().(*buffer)
	buf.ref()
	return buf
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
