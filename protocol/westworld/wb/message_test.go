package wb

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestReadWrite(t *testing.T) {
	data := []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06}
	buffer := make([]byte, 64*1024)
	wm, err := NewData(1, data, buffer)
	assert.Nil(t, err)
	assert.Equal(t, int32(1), wm.Sequence)
	assert.Equal(t, DATA, wm.Type)
	assert.Equal(t, int32(-1), wm.Ack)
	assert.Equal(t, data, wm.Data)

	 wm2, err := FromBuffer(wm.ToBuffer())
	 assert.Nil(t, err)
	 assert.Equal(t, wm.Sequence, wm2.Sequence)
	 assert.Equal(t, wm.Type, wm2.Type)
	 assert.Equal(t, wm.Ack, wm2.Ack)
	 assert.Equal(t, wm.Data, wm2.Data)
}
