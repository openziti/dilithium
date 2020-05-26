package util

import (
	"github.com/pkg/errors"
)

type ByteWriter struct {
	buffer []byte
	pos    int
	len    int
}

func NewByteWriter(buffer []byte) *ByteWriter {
	return &ByteWriter{buffer: buffer, pos: 0, len: len(buffer)}
}

func (self *ByteWriter) Write(p []byte) (n int, err error) {
	if self.pos+len(p) > self.len {
		return 0, errors.New("short")
	}
	n = 0
	for i := 0; self.pos < self.len && i < len(p); i++ {
		self.buffer[self.pos] = p[i]
		self.pos++
		n++
	}
	return n, nil
}

func (self *ByteWriter) Len() int {
	return self.pos
}
