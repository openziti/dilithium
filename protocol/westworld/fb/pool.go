package fb

import (
	flatbuffers "github.com/google/flatbuffers/go"
	"sync"
)

func NewPool() *sync.Pool {
	pool := new(sync.Pool)
	pool.New = func() interface{} {
		return flatbuffers.NewBuilder(64 * 1024)
	}
	return pool
}