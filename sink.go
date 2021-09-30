package dilithium

import (
	"bytes"
	"github.com/pkg/errors"
	"io"
	"sync"
)

type Sink interface {
	Accept(data []byte) error
	Close()
}

func NewReadSinkAdapter(profile WestworldAlgorithm) Sink {
	result := &ReadSinkAdapter{
		reads: make(chan *RxRead, profile.Profile().ReadsQueueSize),
	}

	result.rawReadPool.New = func() interface{} {
		return make([]byte, profile.Profile().PoolBufferSize)
	}

	return result
}

type ReadSinkAdapter struct {
	reads       chan *RxRead
	readBuffer  bytes.Buffer
	rawReadPool sync.Pool
}

func (self *ReadSinkAdapter) Accept(data []byte) error {
	buf := self.rawReadPool.Get().([]byte)
	n := copy(buf, data)
	self.reads <- &RxRead{buf, n, false}
	return nil
}

func (self *ReadSinkAdapter) Close() {
	// TODO: Add timeout
	self.reads <- &RxRead{nil, 0, true}
}

func (self *ReadSinkAdapter) Read(p []byte) (int, error) {
preread:
	for {
		select {
		case read, ok := <-self.reads:
			if !ok {
				return 0, io.EOF
			}
			if !read.Eof {
				n, err := self.readBuffer.Write(read.Buf[:read.Size])
				if err != nil {
					return 0, errors.Wrap(err, "buffer")
				}
				if n != read.Size {
					return 0, errors.New("short buffer")
				}
			} else {
				close(self.reads)
				return 0, io.EOF
			}

		default:
			break preread
		}
	}
	if self.readBuffer.Len() > 0 {
		return self.readBuffer.Read(p)

	} else {
		read, ok := <-self.reads
		if !ok {
			return 0, io.EOF
		}
		if !read.Eof {
			n, err := self.readBuffer.Write(read.Buf[:read.Size])
			if err != nil {
				return 0, errors.Wrap(err, "buffer")
			}
			if n != read.Size {
				return 0, errors.Wrap(err, "short buffer")
			}
			self.rawReadPool.Put(read.Buf)

			return self.readBuffer.Read(p)

		} else {
			close(self.reads)
			return 0, io.EOF
		}
	}
}
