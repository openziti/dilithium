package util

import (
	"bytes"
	"encoding/binary"
)

func ReadInt32(data []byte) (ret int32, err error) {
	buf := bytes.NewBuffer(data)
	err = binary.Read(buf, binary.LittleEndian, &ret)
	return
}
