package loop

import (
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"io"
	"net"
)

type Receiver struct {
	pool *Pool
	conn net.Conn
	Done chan struct{}
}

func NewReceiver(pool *Pool, conn net.Conn) *Receiver {
	return &Receiver{
		pool: pool,
		conn: conn,
		Done: make(chan struct{}),
	}
}

func (self *Receiver) Run() {
	logrus.Info("starting")
	defer logrus.Info("exiting")

	if err := self.receiveStart(); err != nil {
		logrus.Errorf("error receiving start (%v)", err)
		return
	}
	if err := self.receiveData(); err != nil {
		logrus.Errorf("error receiving data (%v)", err)
		return
	}
	close(self.Done)
}

func (self *Receiver) receiveStart() error {
	h, err := readHeader(self.conn, self.pool)
	if err != nil {
		return err
	}
	if h.mt != START {
		return errors.Errorf("expected start message")
	}
	if h.sz != 0 {
		return errors.Errorf("non-empty start message")
	}
	h.buffer.unref()
	return nil
}

func (self *Receiver) receiveData() error {
	count := 0
	for {
		h, err := readHeader(self.conn, self.pool)
		if err != nil {
			return err
		}
		if h.mt == DATA {
			logrus.Infof("reading block #%d", count)
			buffer := self.pool.get()
			n, err := io.ReadFull(self.conn, buffer.data[0:h.sz])
			if err != nil {
				return err
			}
			if n != int(h.sz) {
				return errors.Errorf("short read [%d != %d]", n, h.sz)
			}
			buffer.unref()
			count++

		} else if h.mt == END {
			if h.sz != 0 {
				return errors.Errorf("non-empty end message")
			}
			return nil

		} else {
			return errors.Errorf("unexpected message type (%d)", h.mt)
		}
	}
}
