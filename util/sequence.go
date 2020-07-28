package util

import (
	"math"
	"sync/atomic"
)

type Sequence struct {
	nextValue int32
}

func NewSequence(nextValue int32) *Sequence {
	return &Sequence{nextValue: nextValue - 1}
}

func (self *Sequence) ResetTo(nextValue int32) {
	atomic.StoreInt32(&self.nextValue, nextValue-1)
}

func (self *Sequence) Next() int32 {
	atomic.CompareAndSwapInt32(&self.nextValue, math.MaxInt32, -1)
	return atomic.AddInt32(&self.nextValue, 1)
}
