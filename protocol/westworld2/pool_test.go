package westworld2

import (
	"log"
	"testing"
	"time"
)

func TestPutGet(t *testing.T) {
	tp := newPool("test", nil)
	total := 1000000
	start := time.Now()
	for i := 0; i < total; i++ {
		buffer := tp.get()
		buffer.unref()
	}
	seconds := time.Since(start).Seconds()
	log.Printf("%d put/get, %.2f seconds, %.2f put/get/sec", total, seconds, float64(total)/seconds)
}
