package loop

import "sync/atomic"

type buffer struct {
	data []byte
	sz   uint32
	pool *pool
	refs int32
}

func newBuffer(pool *pool) *buffer {
	return &buffer{
		data: make([]byte, pool.sz),
		pool: pool,
		refs: 0,
	}
}

func (self *buffer) ref() {
	atomic.AddInt32(&self.refs, 1)
}

func (self *buffer) unref() {
	if atomic.AddInt32(&self.refs, -1) < 1 {
		self.sz = 0
		self.pool.put(self)
	}
}
