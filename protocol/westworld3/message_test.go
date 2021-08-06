package westworld3

import (
	"encoding/hex"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func init() {
	for i := 0; i < 16*1024; i++ {
		wireMessageBenchmarkData[i] = uint8(i)
	}
	wireMessageBenchmarkPool = newPool("wireMessageBenchmark", dataStart+(16*1024), NewNilInstrument().NewInstance("", nil))
}

var wireMessageBenchmarkData [16 * 1024]byte
var wireMessageBenchmarkPool *pool

func TestHello(t *testing.T) {
	p := newPool("test", 1024, NewNilInstrument().NewInstance("", nil))
	wm, err := newHello(11, hello{protocolVersion, 6}, nil, p)
	assert.NoError(t, err)
	fmt.Println(hex.Dump(wm.buffer.data[:wm.buffer.uz]))
	assert.Equal(t, uint32(dataStart+5), wm.buffer.uz)

	wmOut, err := decodeHeader(wm.buffer)
	assert.NoError(t, err)
	h, a, err := wmOut.asHello()
	assert.NoError(t, err)
	assert.Equal(t, int32(11), wmOut.seq)
	assert.Equal(t, HELLO, wmOut.messageType())
	assert.Equal(t, protocolVersion, h.version)
	assert.Equal(t, uint8(6), h.profile)
	assert.Equal(t, 0, len(a))
}

func TestHelloResponse(t *testing.T) {
	p := newPool("test", 1024, NewNilInstrument().NewInstance("", nil))
	wm, err := newHello(12, hello{protocolVersion, 6}, &Ack{11, 11}, p)
	assert.NoError(t, err)
	fmt.Println(hex.Dump(wm.buffer.data[:wm.buffer.uz]))
	assert.Equal(t, uint32(dataStart+4+5), wm.buffer.uz)
	assert.True(t, wm.hasFlag(INLINE_ACK))

	wmOut, err := decodeHeader(wm.buffer)
	assert.NoError(t, err)
	h, a, err := wmOut.asHello()
	assert.NoError(t, err)
	assert.Equal(t, int32(12), wmOut.seq)
	assert.Equal(t, HELLO, wmOut.messageType())
	assert.Equal(t, protocolVersion, h.version)
	assert.Equal(t, uint8(6), h.profile)
	assert.Equal(t, 1, len(a))
	assert.Equal(t, int32(11), a[0].Start)
	assert.Equal(t, int32(11), a[0].End)
}

func TestAck(t *testing.T) {
	p := newPool("test", 1024, NewNilInstrument().NewInstance("", nil))
	rtt := uint16(332)
	wm, err := newAck([]Ack{{1, 1}, {3, 5}}, 10240, &rtt, p)
	assert.NoError(t, err)
	fmt.Println(hex.Dump(wm.buffer.data[:wm.buffer.uz]))

	wmOut, err := decodeHeader(wm.buffer)
	assert.NoError(t, err)
	a, rxPortalSz, rttOut, err := wmOut.asAck()
	assert.NoError(t, err)
	assert.Equal(t, int32(-1), wmOut.seq)
	assert.Equal(t, ACK, wmOut.messageType())
	assert.Equal(t, 2, len(a))
	assert.Equal(t, int32(1), a[0].Start)
	assert.Equal(t, int32(1), a[0].End)
	assert.Equal(t, int32(3), a[1].Start)
	assert.Equal(t, int32(5), a[1].End)
	assert.Equal(t, int32(10240), rxPortalSz)
	assert.NotNil(t, rttOut)
	assert.Equal(t, rtt, *rttOut)
}

func TestAckNoRTT(t *testing.T) {
	p := newPool("test", 1024, NewNilInstrument().NewInstance("", nil))
	wm, err := newAck([]Ack{{63, 64}}, 0, nil, p)
	assert.NoError(t, err)
	fmt.Println(hex.Dump(wm.buffer.data[:wm.buffer.uz]))

	wmOut, err := decodeHeader(wm.buffer)
	assert.NoError(t, err)
	a, rxPortalSz, rttOut, err := wmOut.asAck()
	assert.NoError(t, err)
	assert.Equal(t, int32(-1), wmOut.seq)
	assert.Equal(t, ACK, wmOut.messageType())
	assert.Equal(t, 1, len(a))
	assert.Equal(t, int32(63), a[0].Start)
	assert.Equal(t, int32(64), a[0].End)
	assert.Equal(t, int32(0), rxPortalSz)
	assert.Nil(t, rttOut)
}

func TestData(t *testing.T) {
	p := newPool("tooSmall", 1024, NewNilInstrument().NewInstance("", nil))
	_, err := newData(64, nil, wireMessageBenchmarkData[:], p)
	assert.Error(t, err)
	p = newPool("test", 24*1024, NewNilInstrument().NewInstance("", nil))
	rttIn := new(uint16)
	*rttIn = 200
	wm, err2 := newData(64, rttIn, wireMessageBenchmarkData[:], p)
	assert.NoError(t, err2)
	fmt.Println(hex.Dump(wm.buffer.data[:wm.buffer.uz]))

	wmOut, err := decodeHeader(wm.buffer)
	assert.NoError(t, err)
	data, rttOut, err := wmOut.asData()
	assert.NoError(t, err)
	assert.Equal(t, rttIn, rttOut)
	assert.EqualValues(t, wireMessageBenchmarkData[:], data)
}

func TestKeepalive(t *testing.T) {
	p := newPool("test", dataStart+4, NewNilInstrument().NewInstance("", nil))
	wm, err := newKeepalive(23411, p)
	assert.NoError(t, err)
	fmt.Println(hex.Dump(wm.buffer.data[:wm.buffer.uz]))

	wmOut, err := decodeHeader(wm.buffer)
	assert.NoError(t, err)
	assert.Equal(t, wm.seq, wmOut.seq)
	assert.Equal(t, KEEPALIVE, wmOut.mt)
	rxPortalSz, err := wmOut.asKeepalive()
	assert.NoError(t, err)
	assert.Equal(t, 23411, rxPortalSz)
}

func TestClose(t *testing.T) {
	p := newPool("test", dataStart, NewNilInstrument().NewInstance("", nil))
	wm, err := newClose(10233, p)
	assert.NoError(t, err)
	fmt.Println(hex.Dump(wm.buffer.data[:wm.buffer.uz]))

	wmOut, err := decodeHeader(wm.buffer)
	assert.NoError(t, err)
	assert.Equal(t, wm.seq, wmOut.seq)
	assert.Equal(t, CLOSE, wmOut.mt)
}

func TestWireMessageInsertData(t *testing.T) {
	p := newPool("test", 1024, NewNilInstrument().NewInstance("", nil))
	wm := &wireMessage{seq: 0, mt: DATA, buffer: p.get()}
	copy(wm.buffer.data[dataStart:], []byte{0x01, 0x02, 0x03, 0x04})
	wmOut, err := wm.encodeHeader(4)
	assert.NoError(t, err)
	assert.Equal(t, wm.buffer.uz, uint32(dataStart+4))
	assert.Equal(t, wm, wmOut)

	err = wm.insertData([]byte{0x0a, 0x0b, 0x0c, 0x0d})
	assert.NoError(t, err)
	assert.Equal(t, wm.buffer.uz, uint32(dataStart+8))
	assert.ElementsMatch(t, []byte{0x0a, 0x0b, 0x0c, 0x0d, 0x01, 0x02, 0x03, 0x04}, wm.buffer.data[dataStart:wm.buffer.uz])
}

func benchmarkWireMessageInsertData(dataSz, insertSz int, b *testing.B) {
	for i := 0; i < b.N; i++ {
		wm := &wireMessage{seq: 0, mt: DATA, buffer: wireMessageBenchmarkPool.get()}
		copy(wm.buffer.data[dataStart:], wireMessageBenchmarkData[:dataSz])
		if _, err := wm.encodeHeader(uint16(dataSz)); err != nil {
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
	p := newPool("test", 1024, NewNilInstrument().NewInstance("", nil))
	wm := &wireMessage{seq: 0, mt: DATA, buffer: p.get()}
	copy(wm.buffer.data[dataStart:], []byte{0x01, 0x02, 0x03, 0x04})
	wmOut, err := wm.encodeHeader(4)
	assert.NoError(t, err)
	assert.Equal(t, wm.buffer.uz, uint32(dataStart+4))
	assert.Equal(t, wm, wmOut)

	err = wm.appendData([]byte{0x0a, 0x0b, 0x0c, 0x0d})
	assert.NoError(t, err)
	assert.Equal(t, wm.buffer.uz, uint32(dataStart+8))
	assert.ElementsMatch(t, []byte{0x01, 0x02, 0x03, 0x04, 0x0a, 0x0b, 0x0c, 0x0d}, wm.buffer.data[dataStart:wm.buffer.uz])
}

func benchmarkWireMessageAppendData(dataSz, insertSz int, b *testing.B) {
	for i := 0; i < b.N; i++ {
		wm := &wireMessage{seq: 0, mt: DATA, buffer: wireMessageBenchmarkPool.get()}
		copy(wm.buffer.data[dataStart:], wireMessageBenchmarkData[:dataSz])
		if _, err := wm.encodeHeader(uint16(dataSz)); err != nil {
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
