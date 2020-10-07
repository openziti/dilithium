package loop

import (
	"bytes"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEncodeDecodeMessage(t *testing.T) {
	pool := NewPool(19)

	inH := &header{0, START, 1024*1024 + 68, pool.get()}
	inH.encode()
	assert.Equal(t, int(inH.buffer.uz), headerSz)

	logrus.Infof("inH buffer data = %v", inH.buffer.data)

	outBuffer := pool.get()
	for i := 0; i < int(inH.buffer.uz); i++ {
		outBuffer.data[i] = inH.buffer.data[i]
	}
	outBuffer.uz = inH.buffer.uz

	logrus.Infof("outH buffer data = %v", outBuffer.data)

	outH, err := decode(outBuffer)
	assert.NoError(t, err)
	assert.Equal(t, inH.seq, outH.seq)
	assert.Equal(t, inH.mt, outH.mt)
	assert.Equal(t, int(inH.sz), int(outH.sz))
}

func TestReadWriteHeader(t *testing.T) {
	pool := NewPool(512)

	inH := &header{0, START, 1024*1024 + 68, pool.get()}
	buf := new(bytes.Buffer)
	err := writeHeader(inH, buf)
	assert.NoError(t, err)
	assert.Equal(t, int(inH.buffer.uz), headerSz)
	assert.Equal(t, headerSz, buf.Len())

	outH, err2 := readHeader(buf, pool)
	if err2 != nil {
		logrus.Infof("error (%v)", err2)
	}
	assert.NoError(t, err2)
	assert.Equal(t, int(pool.Allocations), 2)

	assert.Equal(t, inH.seq, outH.seq)
	assert.Equal(t, inH.mt, outH.mt)
	assert.Equal(t, int(inH.sz), int(outH.sz))
}

func TestReadWriteHeaderMulti(t *testing.T) {
	pool := NewPool(512)
	inH := &header{0, START, 0, pool.get()}
	buf := new(bytes.Buffer)
	err := writeHeader(inH, buf)
	assert.Equal(t, int(inH.buffer.uz), headerSz)
	assert.NoError(t, err)
	err = writeHeader(inH, buf)
	assert.NoError(t, err)
	outH, err2 := readHeader(buf, pool)
	assert.NoError(t, err2)
	assert.Equal(t, inH.seq, outH.seq)
	assert.Equal(t, inH.mt, outH.mt)
	assert.Equal(t, inH.sz, outH.sz)
}
