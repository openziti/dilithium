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
		return 0, errors.New("short2")
	}
	n = copy(self.buffer[self.pos:self.len], p)
	self.pos += n
	if n != len(p) {
		return 0, errors.Errorf("short [%d != %d]", n, len(p))
	}
	return n, nil
}

func (self *ByteWriter) Len() int {
	return self.pos
}
