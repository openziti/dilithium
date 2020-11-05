package loop

import (
	"crypto/sha512"
	"github.com/openziti/dilithium/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"math/rand"
	"time"
)

type dataBlock struct {
	buffer *buffer
}

func encodeDataBlock(b *buffer) (*buffer, error) {
	start := time.Now()
	last := start
	for i := 2 + 64; i < int(b.sz); i++ {
		b.data[i] = byte(rand.Intn(255))
		if time.Since(start).Milliseconds() > 5000 {
			logrus.Infof("generating random payload (%0.2f%%)", (float32(i)/float32(b.sz))*100.0)
			start = time.Now()
		}
	}
	if time.Since(last).Milliseconds() > 5000 {
		logrus.Infof("hashing payload...")
	}
	hash := sha512.Sum512(b.data[dataHeaderSz:])
	copy(b.data[2:], hash[:])
	util.WriteUint16(b.data, 64)
	b.uz = b.sz
	return b, nil
}

func decodeDataBlock(b *buffer) ([]byte, []byte, error) {
	if b.uz < 4 {
		return nil, nil, errors.Errorf("buffer too small [at least 4 required]")
	}
	hashSz := util.ReadUint16(b.data[0:2])
	if b.uz < int64(2+hashSz) {
		return nil, nil, errors.Errorf("buffer too small [%d current, at least %d required]", b.uz, 4+hashSz)
	}
	return b.data[2 : 2+hashSz], b.data[2+hashSz:], nil
}

type DataSet struct {
	blocks []*buffer
	pool   *Pool
}

func NewDataSet(sz int64) (*DataSet, error) {
	ds := &DataSet{
		pool: NewPool(dataHeaderSz + sz),
	}

	rand.Seed(time.Now().UnixNano())
	b, err := encodeDataBlock(ds.pool.get())
	if err != nil {
		return nil, err
	}
	ds.blocks = append(ds.blocks, b)

	return ds, nil
}

const hashSizeSz = 2
const sha512Sz = 64
const dataHeaderSz = hashSizeSz + sha512Sz
