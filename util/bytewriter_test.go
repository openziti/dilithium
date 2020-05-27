package util

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMultipleWrites(t *testing.T) {
	s0 := []byte{ 0x01, 0x02, 0x03, 0x04 }
	s1 := []byte{ 0x0a, 0x0b, 0x0c, 0x0d }
	trg := make([]byte, 8)
	bw := NewByteWriter(trg)

	n, err := bw.Write(s0)
	assert.Nil(t, err)
	assert.Equal(t, len(s0), n)

	n, err = bw.Write(s1)
	assert.Nil(t, err)
	assert.Equal(t, len(s1), n)

	assert.Equal(t, s0, trg[0:4])
	assert.Equal(t, s1, trg[4:8])
}
