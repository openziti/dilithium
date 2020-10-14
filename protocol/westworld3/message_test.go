package westworld3

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestWireMessageInsertData(t *testing.T) {
	p := newPool("test", 1024, nil)
	wm := &wireMessage{seq: 0, mt: DATA, data: []byte{0x01, 0x02, 0x03, 0x04}, buffer: p.get()}
	wmOut, err := wm.encode()
	assert.NoError(t, err)
	assert.Equal(t, wm.buffer.uz, uint32(headerSz+4))
	assert.Equal(t, wm, wmOut)

	err = wm.insertData([]byte{0x0a, 0x0b, 0x0c, 0x0d})
	assert.NoError(t, err)
	assert.Equal(t, wm.buffer.uz, uint32(headerSz+8))
	assert.ElementsMatch(t, []byte{0x0a, 0x0b, 0x0c, 0x0d, 0x01, 0x02, 0x03, 0x04}, wm.buffer.data[headerSz:wm.buffer.uz])
}
