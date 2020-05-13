package pb

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestReadWrite(t *testing.T) {
	wmIn := NewHello(9, "OH, WOW")

	buffer := new(bytes.Buffer)

	err := WriteMessage(wmIn, buffer)
	assert.Nil(t, err)

	wmOut, err := ReadMessage(buffer)
	assert.Nil(t, err)

	assert.Equal(t, wmIn.Type, wmOut.Type)
	assert.Equal(t, wmIn.HelloPayload.Session, wmOut.HelloPayload.Session)
}