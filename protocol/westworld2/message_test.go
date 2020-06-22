package westworld2

import (
	"github.com/stretchr/testify/assert"
	"log"
	"math/rand"
	"testing"
	"time"
)

func TestRewriteAck(t *testing.T) {
	testPool := newPool("test", nil)
	data, _ := makeData(16*1024, 0)

	wm := newData(99, data, testPool)
	assert.Equal(t, int32(-1), wm.ack)
	assert.Equal(t, []byte{0xff, 0xff, 0xff, 0xff}, wm.buffer.data[5:9])

	wm.rewriteAck(8)
	assert.Equal(t, int32(8), wm.ack)
	assert.Equal(t, []byte{0x00, 0x00, 0x00, 0x08}, wm.buffer.data[5:9])

	wm.buffer.unref()
}

func TestMillionsOfMessages(t *testing.T) {
	maxDataLen := 16 * 1024
	sizeVariations := 1024
	data, szs := makeData(maxDataLen, sizeVariations)

	testPool := newPool("test", nil)

	cycles := 1000000
	start := time.Now()
	for i := 0; i < cycles; i++ {
		szi := i % 1024
		sz := szs[szi]
		wmin := newData(int32(i), data[:sz], testPool)
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
	log.Printf("%d messages, %.2f seconds. %.2f messages/sec", cycles, seconds, float64(cycles)/seconds)
}

func makeData(sz int, variations int) (data []byte, szs []int16) {
	rand.Seed(time.Now().UnixNano())
	if sz > 0 {
		data = make([]byte, sz)
		for i := 0; i < sz; i++ {
			data[i] = byte(rand.Int())
		}
	}
	if variations > 0 {
		szs = make([]int16, variations)
		for i := 0; i < 1024; i++ {
			szs[i] = int16(rand.Intn(sz))
		}
	}
	return
}

func TestReadWriteInt64(t *testing.T) {
	buf := make([]byte, 8)
	now := time.Now().UnixNano()
	WriteInt64(buf, now)
	later := ReadInt64(buf)
	assert.Equal(t, now, later)
}
