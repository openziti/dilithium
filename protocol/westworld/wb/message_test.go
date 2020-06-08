package wb

import (
	"encoding/binary"
	"github.com/michaelquigley/dilithium/util"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestReadWrite(t *testing.T) {
	pool := NewBufferPool("test")
	data := []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06}
	wm, err := NewData(1, data, pool)
	assert.Nil(t, err)
	assert.Equal(t, int32(1), wm.Sequence)
	assert.Equal(t, DATA, wm.Type)
	assert.Equal(t, int32(-1), wm.Ack)
	assert.Equal(t, data, wm.Data)

	wm2, err := FromBuffer(wm.ToBuffer(), nil)
	assert.Nil(t, err)
	assert.Equal(t, wm.Sequence, wm2.Sequence)
	assert.Equal(t, wm.Type, wm2.Type)
	assert.Equal(t, wm.Ack, wm2.Ack)
	assert.Equal(t, wm.Data, wm2.Data)
}

func TestRewriteAck(t *testing.T) {
	pool := NewBufferPool("test")
	data := make([]byte, 128)
	for i := 0; i < len(data); i++ {
		data[i] = uint8(i)
	}

	ackData := make([]byte, 4)
	writer := util.NewByteWriter(ackData)
	err := binary.Write(writer, binary.LittleEndian, int32(-1))
	assert.Nil(t, err)

	wm, err := NewData(1, data, pool)
	assert.Nil(t, err)
	assert.Equal(t, ackData, wm.buffer[5:9])

	err = wm.RewriteAck(99)
	assert.Nil(t, err)
	assert.Equal(t, int32(99), wm.Ack)

	writer = util.NewByteWriter(ackData)
	err = binary.Write(writer, binary.LittleEndian, int32(99))
	assert.Nil(t, err)

	assert.Equal(t, ackData, wm.buffer[5:9])

	wm2, err := FromBuffer(wm.buffer, pool)
	assert.Nil(t, err)
	assert.Equal(t, wm.Sequence, wm2.Sequence)
	assert.Equal(t, wm.Type, wm2.Type)
	assert.Equal(t, wm.Ack, wm2.Ack)
	assert.Equal(t, wm.Data, wm2.Data)
}
