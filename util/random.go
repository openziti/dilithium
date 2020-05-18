package util

import (
	"math/rand"
	"time"
)

func init() {
	r = rand.New(rand.NewSource(time.Now().UnixNano()))
}

var r *rand.Rand

func RandomSequence() int32 {
	return r.Int31n(1024)
}