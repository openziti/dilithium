package loop

import (
	"github.com/michaelquigley/dilithium/util"
	"github.com/pkg/errors"
	"io"
	"net"
)

type header struct {
	seq    uint32
	mt     messageType
	sz     uint32
	buffer *buffer
}

func (self *header) encode() error {
	if self.buffer.sz < headerSz {
		return errors.Errorf("short buffer [%d < %d]", self.buffer.sz, headerSz)
	}

	for i := 0; i < 4; i++ {
		self.buffer.data[i] = magick[i]
	}
	util.WriteUint32(self.buffer.data[4:9], self.seq)
	self.buffer.data[9] = byte(self.mt)
	util.WriteUint32(self.buffer.data[10:14], self.sz)
	self.buffer.uz = headerSz

	return nil
}

func decode(buffer *buffer) (*header, error) {
	if buffer.uz != headerSz {
		return nil, errors.Errorf("short buffer [%d != %d]", buffer.sz, headerSz)
	}
	for i := 0; i < 4; i++ {
		if buffer.data[i] != magick[i] {
			return nil, errors.Errorf("invalid magick")
		}
	}
	h := &header{
		seq:    util.ReadUint32(buffer.data[4:9]),
		mt:     messageType(buffer.data[9]),
		sz:     util.ReadUint32(buffer.data[10:14]),
		buffer: buffer,
	}
	return h, nil
}

func readHeader(conn net.Conn, pool *pool) (h *header, data *buffer, err error) {
	hBuffer := pool.get()
	hData := hBuffer.data[0:14]
	n, err := io.ReadFull(conn, hData)
	if n != headerSz {
		hBuffer.unref()
		return nil, nil, errors.Errorf("weird read [%d != %d]", n, headerSz)
	}
	if err != nil {
		hBuffer.unref()
		return nil, nil, errors.Wrap(err, "error reading header")
	}
	inH, err := decode(hBuffer)
	if err != nil {
		hBuffer.unref()
		return nil, nil, errors.Wrap(err, "error decoding header buffer")
	}
	h = inH
	return
}

func writeHeader(h *header, conn net.Conn) error {
	if err := h.encode(); err != nil {
		return errors.Wrap(err, "error encoding buffer")
	}
	n, err := conn.Write(h.buffer.data[:h.buffer.uz])
	if uint32(n) != h.buffer.uz {
		return errors.Errorf("short header write [%d != %d]", n, h.buffer.uz)
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

var magick = [4]byte{0xCA, 0xFE, 0xBA, 0xBE}

const headerSz = 13
