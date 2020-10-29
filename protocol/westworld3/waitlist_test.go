package westworld3

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestArrayWaitlist_Add_Next(t *testing.T) {
	aw := &arrayWaitlist{}
	deadline := time.Now().Add(200 * time.Millisecond)
	aw.Add(&wireMessage{seq: int32(99)}, deadline)

	wmOut, deadlineOut := aw.Next()
	assert.NotNil(t, wmOut)
	assert.Equal(t, int32(99), wmOut.seq)
	assert.Equal(t, deadline, deadlineOut)

	wmOut, deadlineOut = aw.Next()
	assert.Nil(t, wmOut)
	assert.Equal(t, time.Time{}, deadlineOut)
}

func TestArrayWaitlist_Add_Remove(t *testing.T) {
	aw := &arrayWaitlist{}
	wm := &wireMessage{seq: int32(66)}
	deadline := time.Now().Add(200 * time.Millisecond)
	aw.Add(wm, deadline)

	aw.Remove(wm)
	wmOut, deadlineOut := aw.Next()
	assert.Nil(t, wmOut)
	assert.Equal(t, time.Time{}, deadlineOut)
}

func benchmarkArrayWaitlist_Add_Next(sz int, b *testing.B) {
	toAdd := make([]*waitlistSubject, 0)
	for i := 0; i < sz; i++ {
		toAdd = append(toAdd, &waitlistSubject{time.Now().Add(200 * time.Millisecond), &wireMessage{seq: int32(i)}})
	}
	aw := &arrayWaitlist{}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for i := 0; i < sz; i++ {
			aw.Add(toAdd[i].wm, toAdd[i].deadline)
		}
		for i := 0; i < sz; i++ {
			aw.Next()
		}
	}
}
func BenchmarkArrayWaitlist_Add_Next_1024(b *testing.B)  { benchmarkArrayWaitlist_Add_Next(1024, b) }
func BenchmarkArrayWaitlist_Add_Next_4096(b *testing.B)  { benchmarkArrayWaitlist_Add_Next(4096, b) }
func BenchmarkArrayWaitlist_Add_Next_16384(b *testing.B) { benchmarkArrayWaitlist_Add_Next(16384, b) }

func benchmarkArrayWaitlist_Add_Remove(sz int, b *testing.B) {
	toAdd := make([]*waitlistSubject, 0)
	for i := 0; i < sz; i++ {
		toAdd = append(toAdd, &waitlistSubject{time.Now().Add(200 * time.Millisecond), &wireMessage{seq: int32(i)}})
	}
	aw := &arrayWaitlist{}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for i := 0; i < sz; i++ {
			aw.Add(toAdd[i].wm, toAdd[i].deadline)
		}
		for i := 0; i < sz; i++ {
			aw.Remove(toAdd[i].wm)
		}
	}
}
func BenchmarkArrayWaitlist_Add_Remove_1024(b *testing.B) {
	benchmarkArrayWaitlist_Add_Remove(1024, b)
}
func BenchmarkArrayWaitlist_Add_Remove_4096(b *testing.B) {
	benchmarkArrayWaitlist_Add_Remove(4096, b)
}
func BenchmarkArrayWaitlist_Add_Remove_16384(b *testing.B) {
	benchmarkArrayWaitlist_Add_Remove(16384, b)
}

func benchmarkArrayWaitlist_Add_Remove_Reverse(sz int, b *testing.B) {
	toAdd := make([]*waitlistSubject, 0)
	for i := 0; i < sz; i++ {
		toAdd = append(toAdd, &waitlistSubject{time.Now().Add(200 * time.Millisecond), &wireMessage{seq: int32(i)}})
	}
	aw := &arrayWaitlist{}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for i := 0; i < sz; i++ {
			aw.Add(toAdd[i].wm, toAdd[i].deadline)
		}
		for i := sz - 1; i >= 0; i-- {
			aw.Remove(toAdd[i].wm)
		}
	}
}
func BenchmarkArrayWaitlist_Add_Remove_Reverse_1024(b *testing.B) {
	benchmarkArrayWaitlist_Add_Remove_Reverse(1024, b)
}
func BenchmarkArrayWaitlist_Add_Remove_Reverse_4096(b *testing.B) {
	benchmarkArrayWaitlist_Add_Remove_Reverse(4096, b)
}
func BenchmarkArrayWaitlist_Add_Remove_Reverse_16384(b *testing.B) {
	benchmarkArrayWaitlist_Add_Remove_Reverse(16384, b)
}
