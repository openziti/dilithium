package westworld2

import (
	"github.com/sirupsen/logrus"
	"sync"
	"sync/atomic"
)

type pool struct {
	name   string
	store  *sync.Pool
	allocs int64
}

func newPool(name string) *pool {
	p := &pool{
		name:  name,
		store: new(sync.Pool),
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
	logrus.WithField("context", self.name).Info("allocate")
	atomic.AddInt64(&self.allocs, 1)
	return newBuffer(self)
}
