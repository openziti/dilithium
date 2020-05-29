package westworld2

import (
	"github.com/stretchr/testify/assert"
	"log"
	"testing"
	"time"
)

func TestPutGet(t *testing.T) {
	tp := newPool("test")
	total := 1000000
	start := time.Now()
	for i := 0; i < total; i++ {
		buffer := tp.get()
		buffer.unref()
	}
	seconds := time.Since(start).Seconds()
	log.Printf("%d put/get, %.2f seconds, %.2f put/get/sec", total, seconds, float64(total)/seconds)
	assert.Equal(t, int64(1), tp.allocs)
}
