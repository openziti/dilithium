package westworld3

import (
	"testing"
	"time"
)

func benchmarkArrayWaitlist(sz int, b *testing.B) {
	var toAdd []*retxSubject
	for i := 0; i < sz; i++ {
		toAdd = append(toAdd, &retxSubject{time.Now().Add(200 * time.Millisecond), &wireMessage{seq: int32(i)}})
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

func benchmarkBtreeWaitlist_Add(sz int, b *testing.B) {
	var toAdd []*retxSubject
	for i := 0; i < sz; i++ {
		toAdd = append(toAdd, &retxSubject{time.Now().Add(200 * time.Millisecond), &wireMessage{seq: int32(i)}})
	}
	btw := newBtreeWaitlist()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for i := 0; i < sz; i++ {
			if err := btw.Add(toAdd[i].wm, toAdd[i].deadline); err != nil {
				panic(err)
			}
		}
		for i := 0; i < sz; i++ {
			btw.Next()
		}
	}
}
func BenchmarkBtreeWaitlist_1024(b *testing.B)  { benchmarkBtreeWaitlist_Add(1024, b) }
func BenchmarkBtreeWaitlist_4096(b *testing.B)  { benchmarkBtreeWaitlist_Add(4096, b) }
func BenchmarkBtreeWaitlist_16384(b *testing.B) { benchmarkBtreeWaitlist_Add(16384, b) }
