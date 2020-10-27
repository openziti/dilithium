package westworld3

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestArrayWaitlist_Add_Next(t *testing.T) {
	aw := &arrayWaitlist{}
	deadline := time.Now().Add(200 * time.Millisecond)
	err := aw.Add(&wireMessage{seq: int32(99)}, deadline)
	assert.NoError(t, err)
	wmOut, deadlineOut := aw.Next()
	assert.NotNil(t, wmOut)
	assert.Equal(t, int32(99), wmOut.seq)
	assert.Equal(t, deadline, deadlineOut)
}

func benchmarkArrayWaitlist(sz int, b *testing.B) {
	toAdd := make([]*waitlistSubject, 0)
	for i := 0; i < sz; i++ {
		toAdd = append(toAdd, &waitlistSubject{time.Now().Add(200 * time.Millisecond), &wireMessage{seq: int32(i)}})
	}
	aw := &arrayWaitlist{}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for i := 0; i < sz; i++ {
			if err := aw.Add(toAdd[i].wm, toAdd[i].deadline); err != nil {
				panic(err)
			}
		}
		for i := 0; i < sz; i++ {
			aw.Next()
		}
	}
}
func BenchmarkArrayWaitlist_1024(b *testing.B)  { benchmarkArrayWaitlist(1024, b) }
func BenchmarkArrayWaitlist_4096(b *testing.B)  { benchmarkArrayWaitlist(4096, b) }
func BenchmarkArrayWaitlist_16384(b *testing.B) { benchmarkArrayWaitlist(16384, b) }