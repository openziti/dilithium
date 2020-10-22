package westworld3

import (
	"testing"
	"time"
)

func BenchmarkArrayWaitlist_Add(b *testing.B) {
	aw := &arrayWaitlist{}
	var toAdd []*retxSubject
	for i := 0; i < 1024; i++ {
		toAdd = append(toAdd, &retxSubject{time.Now().Add(200 * time.Millisecond), &wireMessage{seq: int32(i)}})
		time.Sleep(1 * time.Millisecond)
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for i := 0; i < 1024; i++ {
			if err := aw.Add(toAdd[i].wm, toAdd[i].deadline); err != nil {
				panic(err)
			}
		}
	}
}