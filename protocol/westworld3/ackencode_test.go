package westworld3

import (
	"encoding/hex"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSingleEqualAck(t *testing.T) {
	acksIn := []ack{{99, 99}}

	data := make([]byte, 4)
	sz, err := encodeAcks(acksIn, data)
	assert.NoError(t, err)
	assert.Equal(t, uint32(4), sz)
	assert.Zero(t, data[0]&ackSeriesMarker)

	fmt.Println(hex.Dump(data))

	acksOut, err := decodeAcks(data)
	assert.NoError(t, err)
	assert.EqualValues(t, acksIn, acksOut)
}

func TestSingleRangeAck(t *testing.T) {
	acksIn := []ack{{1, 112}}

	data := make([]byte, 1+8)
	sz, err := encodeAcks(acksIn, data)
	assert.NoError(t, err)
	assert.Equal(t, uint32(1+8), sz)
	assert.Equal(t, ackSeriesMarker, data[0]&ackSeriesMarker)

	fmt.Println(hex.Dump(data))

	acksOut, err := decodeAcks(data)
	assert.NoError(t, err)
	assert.EqualValues(t, acksIn, acksOut)
}

func TestSingleRangeSingle(t *testing.T) {
	acksIn := []ack{{66, 66}, {69, 99}, {111, 111}}

	data := make([]byte, 1+4+8+4)
	sz, err := encodeAcks(acksIn, data)
	assert.NoError(t, err)
	assert.Equal(t, uint32(1+4+8+4), sz)
	assert.Equal(t, ackSeriesMarker, data[0]&ackSeriesMarker)

	fmt.Println(hex.Dump(data))

	acksOut, err := decodeAcks(data)
	assert.NoError(t, err)
	assert.EqualValues(t, acksIn, acksOut)
}
