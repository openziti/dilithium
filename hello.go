package dilithium

import (
	"github.com/openziti/dilithium/util"
	"github.com/pkg/errors"
)

type hello struct {
	version uint32
}

func encodeHello(hello hello, data []byte) (n uint32, err error) {
	dataSize := len(data)
	if dataSize < 4 {
		return 0, errors.Errorf("hello too large [%d < 4]", dataSize)
	}
	util.WriteUint32(data, hello.version)
	return 4, nil
}

func decodeHello(data []byte) (hello, uint32, error) {
	dataSize := len(data)
	if dataSize < 5 {
		return hello{}, 0, errors.Errorf("short hello decode buffer [%d < 4]", dataSize)
	}
	return hello{util.ReadUint32(data)}, 4, nil
}
