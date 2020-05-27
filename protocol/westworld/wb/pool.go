package wb

import (
	"github.com/sirupsen/logrus"
	"sync"
)

type BufferPool struct {
	ctx  string
	pool *sync.Pool
}

func NewBufferPool(ctx string) *BufferPool {
	pool := &BufferPool{
		ctx: ctx,
		pool: new(sync.Pool),
	}
	pool.pool.New = pool.Allocate
	return pool
}

func (self *BufferPool) Get() interface{} {
	return self.pool.Get()
}

func (self *BufferPool) Put(x interface{}) {
	self.pool.Put(x)
}

func (self *BufferPool) Allocate() interface{} {
	logrus.WithField("ctx", self.ctx).Info("allocating")
	return make([]byte, 64*1024)
}
