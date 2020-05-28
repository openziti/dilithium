package util

import (
	"github.com/stretchr/testify/assert"
	"log"
	"testing"
)

func TestMultipleWrites(t *testing.T) {
	s0 := []byte{0x01, 0x02, 0x03, 0x04}
	s1 := []byte{0x0a, 0x0b, 0x0c, 0x0d}
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

func TestOverwrite(t *testing.T) {
	s0 := []byte{0x01, 0x02, 0x03, 0x04}
	log.Printf("s0 = %v\n", s0)

	bw := NewByteWriter(s0[1:3])
	n, err := bw.Write([]byte{0xff, 0xfe})
	assert.Nil(t, err)
	assert.Equal(t, 2, n)
	assert.Equal(t, uint8(0xff), s0[1])
	assert.Equal(t, uint8(0xfe), s0[2])

	log.Printf("s0 = %v\n", s0)
}
