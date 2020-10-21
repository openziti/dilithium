package westworld3

import (
	"fmt"
	"testing"
)

func TestProfileDump(t *testing.T) {
	p := NewBaselineProfile()
	fmt.Println(p.Dump())
}
