package westworld2

import "sync/atomic"

type buffer struct {
	data []byte
	used uint16
	pool *pool
	refs int32
}

func newBuffer(pool *pool) *buffer {
	return &buffer{
		data: pool.get(),
		pool: pool,
		refs: 1,
	}
}

func (self* buffer) view() []byte {
	return self.data[:self.used]
}

func (self *buffer) ref() {
	atomic.AddInt32(&self.refs, 1)
}

func (self *buffer) unref() {
	atomic.AddInt32(&self.refs, -1)
	if atomic.LoadInt32(&self.refs) < 1{
		self.pool.put(self.data)
		self.data = nil
		self.used = 0
	}
}