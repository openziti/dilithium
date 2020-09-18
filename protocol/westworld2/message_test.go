package westworld2

import (
	"github.com/michaelquigley/dilithium/util"
	"github.com/stretchr/testify/assert"
	"log"
	"math/rand"
	"testing"
	"time"
)

func TestAckRxPortalSz(t *testing.T) {
	testPool := newPool("test", nil)
	wm := newAck(10, 123456, testPool)
	assert.Equal(t, 4, len(wm.data))
	assert.Equal(t, int(headerSz+4), int(wm.buffer.sz))
}

func TestRewriteAck(t *testing.T) {
	testPool := newPool("test", nil)
	data, _ := makeData(16*1024, 0)

	wm := newData(99, data, testPool)
	assert.Equal(t, int32(-1), wm.ack)
	assert.Equal(t, []byte{0xff, 0xff, 0xff, 0xff}, wm.buffer.data[6:10])

	wm.rewriteAck(8)
	assert.Equal(t, int32(8), wm.ack)
	assert.Equal(t, []byte{0x00, 0x00, 0x00, 0x08}, wm.buffer.data[6:10])

	wm.buffer.unref()
}

func TestReadWriteRtt(t *testing.T) {
	tp := newPool("tp", nil)
	data, _ := makeData(16*1024, 0)

	wm := newData(101, data, tp)
	assert.Equal(t, wm.data, data)

	rtt := time.Now().UnixNano()
	wm.writeRtt(rtt)
	assert.Equal(t, RTT, wm.mf)

	wm2, err := decode(wm.buffer)
	assert.Nil(t, err)
	assert.Equal(t, wm.seq, wm2.seq)
	assert.Equal(t, wm.mt, wm2.mt)
	assert.Equal(t, wm.mf, wm2.mf)
	assert.Equal(t, wm.ack, wm2.ack)
	assert.Equal(t, wm.data, wm2.data)

	rtt2, err := wm2.readRtt()
	assert.Nil(t, err)
	assert.Equal(t, rtt, rtt2)
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
		assert.Equal(t, messageFlag(0), wmout.mf)
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
	util.WriteInt64(buf, now)
	later := util.ReadInt64(buf)
	assert.Equal(t, now, later)
}
