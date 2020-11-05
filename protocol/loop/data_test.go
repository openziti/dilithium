package loop

import (
	"github.com/openziti/dilithium/util"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDataBlockEncodeDecode(t *testing.T) {
	pool := NewPool(dataHeaderSz + (1024 * 1024))

	ds, err := NewDataSet(1024 * 1024)
	assert.NoError(t, err)
	inBuffer := ds.blocks[0]

	outBuffer := pool.get()
	for i := int64(0); i < inBuffer.uz; i++ {
		outBuffer.data[i] = inBuffer.data[i]
	}
	outBuffer.uz = inBuffer.uz

	hash, data, err2 := decodeDataBlock(outBuffer)
	assert.NoError(t, err2)
	assert.EqualValues(t, inBuffer.data[2:2+64], hash)
	assert.EqualValues(t, inBuffer.data[2+64:], data)
	assert.Equal(t, 64, int(util.ReadUint16(inBuffer.data[:2])))
}
