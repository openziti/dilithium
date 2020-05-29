package westworld2

import (
	"github.com/sirupsen/logrus"
	"sync"
)

type pool struct {
	name  string
	store *sync.Pool
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
	return newBuffer(self)
}
