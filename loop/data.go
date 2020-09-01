package loop

import (
	"crypto/sha512"
	"math/rand"
	"time"
)

type dataBlock struct {
	data []byte
	hash []byte
}

type dataSet struct {
	srcData []byte
	block   *dataBlock
}

func newDataSet(sz int) *dataSet {
	ds := &dataSet{
		srcData: make([]byte, sz),
	}

	rand.Seed(time.Now().UnixNano())
	for i := 0; i < sz; i++ {
		ds.srcData[i] = byte(rand.Intn(255))
	}

	hash := sha512.Sum512(ds.srcData)
	ds.block = &dataBlock{
		data: ds.srcData,
		hash: hash[:],
	}

	return ds
}
