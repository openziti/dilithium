package main

import (
	"fmt"
	"math/rand"
	"time"
)

func testTwoByteTimestamps() {
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < 1024; i++ {
		sourceTs := uint16(time.Now().UnixNano() / int64(time.Millisecond))
		randMs := rand.Intn(48000)
		time.Sleep(time.Duration(randMs) * time.Millisecond)
		checkTs := uint16(time.Now().UnixNano() / int64(time.Millisecond))
		delta := checkTs - sourceTs
		if int(delta) < (randMs-int(float64(randMs)*0.05)) || int(delta) > (randMs+int(float64(randMs)*0.05)) {
			fmt.Printf("{%d != %d}", delta, randMs)
		} else {
			fmt.Printf(".")
		}
	}
	fmt.Println()
}
