package westworld2

import "sync/atomic"

type buffer struct {
	data []byte
	sz   uint16
	pool *pool
	refs int32
}

func newBuffer(pool *pool) *buffer {
	return &buffer{
		data: make([]byte, 64*1024),
		pool: pool,
		refs: 1,
	}
}

func (self *buffer) view() []byte {
	return self.data[:self.sz]
}

func (self *buffer) ref() {
	atomic.AddInt32(&self.refs, 1)
}

func (self *buffer) unref() {
	atomic.AddInt32(&self.refs, -1)
	if atomic.LoadInt32(&self.refs) < 1 {
		self.sz = 0
		self.pool.put(self)
	}
}
