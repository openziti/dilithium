package westworld3

import (
	"testing"
	"time"
)

var waitlistBenchmarkToAdd []*retxSubject
func init() {
	for i := 0; i < 1024; i++ {
		waitlistBenchmarkToAdd = append(waitlistBenchmarkToAdd, &retxSubject{time.Now().Add(200 * time.Millisecond), &wireMessage{seq: int32(i)}})
		time.Sleep(1 * time.Millisecond)
	}
}

func BenchmarkArrayWaitlist_Add(b *testing.B) {
	aw := &arrayWaitlist{}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for i := 0; i < 1024; i++ {
			if err := aw.Add(waitlistBenchmarkToAdd[i].wm, waitlistBenchmarkToAdd[i].deadline); err != nil {
				panic(err)
			}
		}
	}
}

func BenchmarkBtreeWaitlist_Add(b *testing.B) {
	btw := newBtreeWaitlist()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for i := 0; i < 1024; i++ {
			if err := btw.Add(waitlistBenchmarkToAdd[i].wm, waitlistBenchmarkToAdd[i].deadline); err != nil {
				panic(err)
			}
		}
	}
}