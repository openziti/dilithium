package westworld2

import (
	"sync"
)

type pool struct {
	name   string
	store  *sync.Pool
	config *Config
}

func newPool(name string, config *Config) *pool {
	p := &pool{
		name:   name,
		store:  new(sync.Pool),
		config: config,
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
	if self.config != nil && self.config.i != nil {
		self.config.i.allocate(self.name)
	}
	return newBuffer(self)
}
