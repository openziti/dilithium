package westworld3

import "sync/atomic"

type buffer struct {
	data []byte
	refs int32
	sz   uint32
	uz   uint32
	pool *pool
}

func newBuffer(pool *pool) *buffer {
	return &buffer{
		data: make([]byte, pool.bufSz),
		pool: pool,
		sz:   pool.bufSz,
		uz:   0,
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
