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

func (self *pool) get() []byte {
	return self.store.Get().([]byte)
}

func (self *pool) put(buffer []byte) {
	self.store.Put(buffer)
}

func (self *pool) allocate() interface{} {
	logrus.WithField("context", self.name).Info("allocate")
	return make([]byte, 64*1024)
}
