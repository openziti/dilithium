package westworld3

import (
	"github.com/openziti/dilithium/util"
	"github.com/pkg/errors"
)

type hello struct {
	version uint32
	profile uint8
}

func encodeHello(hello hello, data []byte) (n uint32, err error) {
	dataSz := len(data)
	if dataSz < 5 {
		return 0, errors.Errorf("hello too large [%d < 5]", dataSz)
	}
	util.WriteUint32(data, hello.version)
	data[4] = hello.profile
	return 5, nil
}

func decodeHello(data []byte) (hello, uint32, error) {
	dataSz := len(data)
	if dataSz < 5 {
		return hello{}, 0, errors.Errorf("short hello buffer [%d < 4]", dataSz)
	}
	return hello{util.ReadUint32(data), data[4]}, 5, nil
}
