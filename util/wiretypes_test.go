package util

import (
	"github.com/stretchr/testify/assert"
	"math"
	"testing"
)

func TestReadWriteUint32(t *testing.T) {
	valueOut := uint32(math.MaxUint32)
	buf := make([]byte, 4)
	WriteUint32(buf, valueOut)
	valueIn := ReadUint32(buf)
	assert.Equal(t, valueOut, valueIn)

	valueOut = 16
	buf = make([]byte, 4)
	WriteUint32(buf, valueOut)
	valueIn = ReadUint32(buf)
	assert.Equal(t, valueOut, valueIn)
}