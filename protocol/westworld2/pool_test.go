package westworld2

import "testing"

func TestPutGet(t *testing.T) {
	tp := newPool("test")
	for i := 0; i < 1000000; i++ {
		buffer := tp.get()
		buffer.unref()
	}
}
