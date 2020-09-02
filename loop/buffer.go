package loop

import "sync/atomic"

type buffer struct {
	data []byte
	uz   uint32
	sz   uint32
	pool *Pool
	refs int32
}

func newBuffer(pool *Pool) *buffer {
	return &buffer{
		data: make([]byte, pool.sz),
		sz:   pool.sz,
		pool: pool,
		refs: 0,
	}
}

func (self *buffer) ref() {
	atomic.AddInt32(&self.refs, 1)
}

func (self *buffer) unref() {
	if atomic.AddInt32(&self.refs, -1) < 1 {
		self.uz = 0
		self.pool.put(self)
	}
}
