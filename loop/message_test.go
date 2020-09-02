package loop

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEncodeDecodeMessage(t *testing.T) {
	pool := NewPool(512)
	inH := &header{0, START, 0, pool.get()}
	inH.encode()
	assert.Equal(t, int(inH.buffer.uz), int(headerSz))
	outH, err := decode(inH.buffer)
	assert.NoError(t, err)
	assert.Equal(t, inH.seq, outH.seq)
	assert.Equal(t, inH.mt, outH.mt)
	assert.Equal(t, inH.sz, outH.sz)
}

func TestReadWriteHeader(t *testing.T) {
	pool := NewPool(512)
	inH := &header{0, START, 0, pool.get()}
	buf := new(bytes.Buffer)
	err := writeHeader(inH, buf)
	assert.Equal(t, int(inH.buffer.uz), int(headerSz))
	assert.NoError(t, err)
	outH, err2 := readHeader(buf, pool)
	assert.NoError(t, err2)
	assert.Equal(t, inH.seq, outH.seq)
	assert.Equal(t, inH.mt, outH.mt)
	assert.Equal(t, inH.sz, outH.sz)
}

func TestReadWriteHeaderMulti(t *testing.T) {
	pool := NewPool(512)
	inH := &header{0, START, 0, pool.get()}
	buf := new(bytes.Buffer)
	err := writeHeader(inH, buf)
	assert.Equal(t, int(inH.buffer.uz), int(headerSz))
	assert.NoError(t, err)
	err = writeHeader(inH, buf)
	assert.NoError(t, err)
	outH, err2 := readHeader(buf, pool)
	assert.NoError(t, err2)
	assert.Equal(t, inH.seq, outH.seq)
	assert.Equal(t, inH.mt, outH.mt)
	assert.Equal(t, inH.sz, outH.sz)
}