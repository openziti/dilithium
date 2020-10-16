package westworld3

import (
	"encoding/hex"
	"fmt"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"testing"
	"time"
)

var sampleAcks []ack
var benchmarkData []byte

func init() {
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < 127; i++ {
		flip := rand.Intn(2)
		if flip == 1 {
			sampleAcks = append(sampleAcks, ack{rand.Int31(), rand.Int31()})
		} else {
			value := rand.Int31()
			sampleAcks = append(sampleAcks, ack{value, value})
		}
	}
	benchmarkData = make([]byte, 1+(127*8))
}

func TestSingleEqualAck(t *testing.T) {
	acksIn := []ack{{99, 99}}

	data := make([]byte, 4)
	sz, err := encodeAcks(acksIn, data)
	assert.NoError(t, err)
	assert.Equal(t, uint32(4), sz)
	assert.Zero(t, data[0]&ackSeriesMarker)

	fmt.Println(hex.Dump(data))

	acksOut, _, err := decodeAcks(data)
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

	acksOut, _, err := decodeAcks(data)
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

	acksOut, _, err := decodeAcks(data)
	assert.NoError(t, err)
	assert.EqualValues(t, acksIn, acksOut)
}

func TestFull127(t *testing.T) {
	data := make([]byte, 1+(127*8)) // max, if all 127 acks in series are ranges
	_, err := encodeAcks(sampleAcks, data)
	assert.NoError(t, err)
	assert.Equal(t, ackSeriesMarker, data[0]&ackSeriesMarker)

	fmt.Println(hex.Dump(data))

	acksOut, _, err := decodeAcks(data)
	assert.NoError(t, err)
	assert.EqualValues(t, sampleAcks, acksOut)
}

func benchmarkAckEncoder(sz int, b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := encodeAcks(sampleAcks[0:sz], benchmarkData)
		if err != nil {
			panic(err)
		}
	}
}
func BenchmarkAckEncoder1(b *testing.B)   { benchmarkAckEncoder(1, b) }
func BenchmarkAckEncoder2(b *testing.B)   { benchmarkAckEncoder(2, b) }
func BenchmarkAckEncoder4(b *testing.B)   { benchmarkAckEncoder(4, b) }
func BenchmarkAckEncoder16(b *testing.B)  { benchmarkAckEncoder(16, b) }
func BenchmarkAckEncoder64(b *testing.B)  { benchmarkAckEncoder(64, b) }
func BenchmarkAckEncoder127(b *testing.B) { benchmarkAckEncoder(127, b) }

func benchmarkAckEncoderDecoder(sz int, b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := encodeAcks(sampleAcks[0:sz], benchmarkData)
		if err != nil {
			panic(err)
		}
		_, _, err = decodeAcks(benchmarkData)
		if err != nil {
			panic(err)
		}
	}
}
func BenchmarkAckEncoderDecoder1(b *testing.B)   { benchmarkAckEncoderDecoder(1, b) }
func BenchmarkAckEncoderDecoder2(b *testing.B)   { benchmarkAckEncoderDecoder(2, b) }
func BenchmarkAckEncoderDecoder4(b *testing.B)   { benchmarkAckEncoderDecoder(4, b) }
func BenchmarkAckEncoderDecoder16(b *testing.B)  { benchmarkAckEncoderDecoder(16, b) }
func BenchmarkAckEncoderDecoder64(b *testing.B)  { benchmarkAckEncoderDecoder(64, b) }
func BenchmarkAckEncoderDecoder127(b *testing.B) { benchmarkAckEncoderDecoder(127, b) }
