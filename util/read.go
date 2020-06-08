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

func ReadUint8(data []byte) (ret uint8, err error) {
	buf := bytes.NewBuffer(data)
	err = binary.Read(buf, binary.LittleEndian, &ret)
	return
}
