package westworld2

import (
	"github.com/stretchr/testify/assert"
	"log"
	"math/rand"
	"testing"
	"time"
)

func TestMillionsOfMessages(t *testing.T) {
	tp := newPool("test")
	data, szs := makeData(16 * 1024)
	start := time.Now()
	total := 1000000
	for i := 0; i < total; i++ {
		szi := i % 1024
		sz := szs[szi]
		wmin := newData(int32(i), data[:sz], tp.get())
		wmout, err := decode(wmin.buffer)
		assert.Nil(t, err)
		assert.Equal(t, int32(i), wmout.seq)
		assert.Equal(t, DATA, wmout.mt)
		assert.Equal(t, int32(-1), wmout.ack)
		assert.Equal(t, sz, int16(len(wmout.data)))
		assert.Equal(t, data[:sz], wmout.data)
		wmout.buffer.unref()
	}
	seconds := time.Since(start).Seconds()
	log.Printf("%d messages, %.2f seconds. %.2f messages/sec", total, seconds, float64(total)/seconds)
}

func makeData(sz int) ([]byte, []int16) {
	rand.Seed(time.Now().UnixNano())
	data := make([]byte, sz)
	for i := 0; i < sz; i++ {
		data[i] = byte(rand.Int())
	}
	szs := make([]int16, 1024)
	for i := 0; i < 1024; i++ {
		szs[i] = int16(rand.Intn(sz))
	}
	return data, szs
}
