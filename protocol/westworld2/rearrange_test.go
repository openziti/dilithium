package westworld2

import (
	"log"
	"testing"
)

func TestRearrange(t *testing.T) {
	data := []byte{ 0, 1, 2, 3, 4 }
	data = append(data[1:], data[0])
	for i, d := range data {
		log.Printf("data[%d] = %d", i, d)
	}
}
