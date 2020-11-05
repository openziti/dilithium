package loop

import (
	"github.com/openziti/dilithium/util"
	"github.com/pkg/errors"
	"io"
)

type header struct {
	seq    uint32
	mt     messageType
	sz     int64
	buffer *buffer
}

func (self *header) encode() error {
	if self.buffer.sz < headerSz {
		return errors.Errorf("short buffer [%d < %d]", self.buffer.sz, headerSz)
	}

	for i := 0; i < 4; i++ {
		self.buffer.data[i] = magick[i]
	}
	util.WriteInt32(self.buffer.data[4:8], int32(self.seq))
	self.buffer.data[8] = byte(self.mt)
	util.WriteInt64(self.buffer.data[9:17], self.sz)
	self.buffer.uz = headerSz

	return nil
}

func decode(buffer *buffer) (*header, error) {
	if buffer.uz != headerSz {
		return nil, errors.Errorf("short buffer [%d != %d]", buffer.uz, headerSz)
	}

	for i := 0; i < 4; i++ {
		if buffer.data[i] != magick[i] {
			return nil, errors.Errorf("invalid magick")
		}
	}
	h := &header{
		seq:    uint32(util.ReadInt32(buffer.data[4:8])),
		mt:     messageType(buffer.data[8]),
		sz:     util.ReadInt64(buffer.data[9:17]),
		buffer: buffer,
	}
	return h, nil
}

func readHeader(r io.Reader, pool *Pool) (*header, error) {
	hBuffer := pool.get()
	n, err := io.ReadFull(r, hBuffer.data[:headerSz])
	if n != headerSz {
		hBuffer.unref()
		return nil, errors.Errorf("weird read [%d != %d]", n, headerSz)
	}
	if err != nil {
		hBuffer.unref()
		return nil, errors.Wrap(err, "error reading header")
	}
	hBuffer.uz = int64(n)
	inH, err2 := decode(hBuffer)
	if err2 != nil {
		hBuffer.unref()
		return nil, errors.Wrap(err2, "error decoding header buffer")
	}
	return inH, nil
}

func writeHeader(h *header, w io.Writer) error {
	if err := h.encode(); err != nil {
		return errors.Wrap(err, "error encoding buffer")
	}
	n, err := w.Write(h.buffer.data[:headerSz])
	if int64(n) != headerSz {
		return errors.Errorf("short header write [%d != %d]", n, headerSz)
	}
	if err != nil {
		return errors.Wrap(err, "error writing header")
	}
	return nil
}

type messageType uint8

const (
	START messageType = iota
	DATA
	END
)

var magick = [4]byte{0xCA, 0xFE, 0xBA, 0xB3}

const headerSz = 17
