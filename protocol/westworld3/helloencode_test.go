package westworld3

import (
	"encoding/hex"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestHelloEncodeDecode(t *testing.T) {
	data := make([]byte, 5)
	sz, err := encodeHello(hello{9006, 0xF}, data)
	assert.NoError(t, err)
	assert.Equal(t, uint32(5), sz)

	fmt.Println(hex.Dump(data))

	outHello, err2 := decodeHello(data)
	assert.NoError(t, err2)
	assert.Equal(t, uint32(9006), outHello.version)
	assert.Equal(t, uint8(0xF), outHello.profile)
}