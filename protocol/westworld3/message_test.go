package westworld3

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func init() {
	for i := 0; i < 16*1024; i++ {
		wireMessageBenchmarkData[i] = uint8(i)
	}
	wireMessageBenchmarkPool = newPool("wireMessageBenchmark", headerSz+(16*1024), nil)
}

var wireMessageBenchmarkData [16 * 1024]byte
var wireMessageBenchmarkPool *pool

func TestWireMessageInsertData(t *testing.T) {
	p := newPool("test", 1024, nil)
	wm := &wireMessage{seq: 0, mt: DATA, data: []byte{0x01, 0x02, 0x03, 0x04}, buffer: p.get()}
	wmOut, err := wm.encode()
	assert.NoError(t, err)
	assert.Equal(t, wm.buffer.uz, uint32(headerSz+4))
	assert.Equal(t, wm, wmOut)

	err = wm.insertData([]byte{0x0a, 0x0b, 0x0c, 0x0d})
	assert.NoError(t, err)
	assert.Equal(t, wm.buffer.uz, uint32(headerSz+8))
	assert.ElementsMatch(t, []byte{0x0a, 0x0b, 0x0c, 0x0d, 0x01, 0x02, 0x03, 0x04}, wm.buffer.data[headerSz:wm.buffer.uz])
}

func benchmarkWireMessageInsertData(dataSz, insertSz int, b *testing.B) {
	for i := 0; i < b.N; i++ {
		wm := &wireMessage{seq: 0, mt: DATA, data: wireMessageBenchmarkData[:dataSz], buffer: wireMessageBenchmarkPool.get()}
		if _, err := wm.encode(); err != nil {
			panic(err)
		}
		if err := wm.insertData(wireMessageBenchmarkData[:insertSz]); err != nil {
			panic(err)
		}
		wm.buffer.unref()
	}
}
func BenchmarkWireMessageInsertData8(b *testing.B)    { benchmarkWireMessageInsertData(8, 8, b) }
func BenchmarkWireMessageInsertData256(b *testing.B)  { benchmarkWireMessageInsertData(256, 8, b) }
func BenchmarkWireMessageInsertData1024(b *testing.B) { benchmarkWireMessageInsertData(1024, 8, b) }
func BenchmarkWireMessageinsertData4096(b *testing.B) { benchmarkWireMessageInsertData(4096, 8, b) }

func TestWireMessageAppendData(t *testing.T) {
	p := newPool("test", 1024, nil)
	wm := &wireMessage{seq: 0, mt: DATA, data: []byte{0x01, 0x02, 0x03, 0x04}, buffer: p.get()}
	wmOut, err := wm.encode()
	assert.NoError(t, err)
	assert.Equal(t, wm.buffer.uz, uint32(headerSz+4))
	assert.Equal(t, wm, wmOut)

	err = wm.appendData([]byte{0x0a, 0x0b, 0x0c, 0x0d})
	assert.NoError(t, err)
	assert.Equal(t, wm.buffer.uz, uint32(headerSz+8))
	assert.ElementsMatch(t, []byte{0x01, 0x02, 0x03, 0x04, 0x0a, 0x0b, 0x0c, 0x0d}, wm.buffer.data[headerSz:wm.buffer.uz])
}

func benchmarkWireMessageAppendData(dataSz, insertSz int, b *testing.B) {
	for i := 0; i < b.N; i++ {
		wm := &wireMessage{seq: 0, mt: DATA, data: wireMessageBenchmarkData[:dataSz], buffer: wireMessageBenchmarkPool.get()}
		if _, err := wm.encode(); err != nil {
			panic(err)
		}
		if err := wm.appendData(wireMessageBenchmarkData[:insertSz]); err != nil {
			panic(err)
		}
		wm.buffer.unref()
	}
}
func BenchmarkWireMessageAppendData8(b *testing.B)    { benchmarkWireMessageAppendData(8, 8, b) }
func BenchmarkWireMessageAppendData256(b *testing.B)  { benchmarkWireMessageAppendData(256, 8, b) }
func BenchmarkWireMessageAppendData1024(b *testing.B) { benchmarkWireMessageAppendData(1024, 8, b) }
func BenchmarkWireMessageAppendData4096(b *testing.B) { benchmarkWireMessageAppendData(4096, 8, b) }

func TestBits(t *testing.T) {
	fmt.Printf("%x\n", 1 << 7)
}