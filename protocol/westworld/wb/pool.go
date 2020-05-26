package wb

import "sync"

func NewBufferPool() *sync.Pool {
	pool := new(sync.Pool)
	pool.New = func() interface{} {
		return make([]byte, 64*1024)
	}
	return pool
}
