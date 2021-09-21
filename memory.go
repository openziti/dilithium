package dilithium

import (
	"sync"
	"sync/atomic"
)

type Buffer struct {
	Data []byte
	Size uint32
	Used uint32
	refs int32
	pool *Pool
}

func NewBuffer(pool *Pool) *Buffer {
	return &Buffer{
		Data: make([]byte, pool.bufSize),
		Size: pool.bufSize,
		Used: 0,
		refs: 0,
		pool: pool,
	}
}

func (buf *Buffer) Ref() {
	atomic.AddInt32(&buf.refs, 1)
}

func (buf *Buffer) Unref() {
	if atomic.AddInt32(&buf.refs, -1) < 1 {
		buf.Used = 0
		//buf.pool.Put(buf)
	}
}

type Pool struct {
	id      string
	bufSize uint32
	store   *sync.Pool
}

func NewPool(id string, bufSize uint32) *Pool {
	pool := &Pool{
		id:      id,
		bufSize: bufSize,
		store:   new(sync.Pool),
	}
	pool.store.New = pool.allocate
	return pool
}

func (pool *Pool) Get() *Buffer {
	buf := pool.store.Get().(*Buffer)
	buf.Ref()
	return buf
}

func (pool *Pool) Put(buf *Buffer) {
	pool.store.Put(buf)
}

func (pool *Pool) allocate() interface{} {
	return NewBuffer(pool)
}
