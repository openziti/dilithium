package loop

import (
	"crypto/sha512"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"io"
	"net"
	"time"
)

type Receiver struct {
	headerPool *Pool
	pool       *Pool
	conn       net.Conn
	blocks     chan *buffer
	blocksDone chan struct{}
	metrics    *Metrics
	rate       *transferReporter
	Done       chan struct{}
}

func NewReceiver(metrics *Metrics, conn net.Conn) *Receiver {
	return &Receiver{
		headerPool: NewPool(headerSz + 1),
		conn:       conn,
		blocks:     make(chan *buffer, 4096),
		blocksDone: make(chan struct{}),
		metrics:    metrics,
		rate:       newTransferReporter("rx"),
		Done:       make(chan struct{}),
	}
}

func (self *Receiver) Run(hasher bool) {
	logrus.Info("starting")
	defer logrus.Info("exiting")

	go self.rate.run()

	if hasher {
		go self.hasher()
		defer func() {
			logrus.Infof("closing hasher")
			close(self.blocks)
			<-self.blocksDone
			close(self.Done)
		}()
	} else {
		defer close(self.Done)
	}

	if err := self.receiveStart(); err != nil {
		logrus.Errorf("error receiving start (%v)", err)
		return
	}
	if err := self.receiveData(hasher); err != nil {
		logrus.Errorf("error receiving data (%v)", err)
		return
	}

	logrus.Infof("[%d] header pool allocations", self.headerPool.Allocations)
	if self.pool != nil {
		logrus.Infof("[%d] data pool allocations", self.pool.Allocations)
	}

	close(self.rate.in)
}

func (self *Receiver) receiveStart() error {
	h, err := readHeader(self.conn, self.headerPool)
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

func (self *Receiver) receiveData(hasher bool) error {
	count := 0
	for {
		h, err := readHeader(self.conn, self.headerPool)
		if err != nil {
			return err
		}
		if h.mt == DATA {
			if self.pool == nil {
				self.pool = NewPool(h.sz)
			}

			buffer := self.pool.get()
			n, err := io.ReadFull(self.conn, buffer.data[0:h.sz])
			if err != nil {
				return err
			}
			if n != int(h.sz) {
				return errors.Errorf("short read [%d != %d]", n, h.sz)
			}
			buffer.uz = int64(n)
			h.buffer.unref()

			self.metrics.Rx(buffer.uz)
			self.rate.in <- &transferReport{time.Now(), buffer.uz}

			if hasher {
				self.blocks <- buffer
			}

			count++

		} else if h.mt == END {
			if h.sz != 0 {
				return errors.Errorf("non-empty end message")
			}
			h.buffer.unref()

			if err := self.sendEnd(); err != nil {
				return err
			}

			return nil

		} else {
			h.buffer.unref()
			return errors.Errorf("unexpected message type (%d)", h.mt)
		}
	}
}

func (self *Receiver) sendEnd() error {
	h := &header{uint32(0), END, 0, self.headerPool.get()}
	if err := writeHeader(h, self.conn); err != nil {
		return err
	}
	h.buffer.unref()
	return nil
}

func (self *Receiver) hasher() {
	logrus.Infof("started")
	defer logrus.Infof("exited")
	defer func() { close(self.blocksDone) }()

	for {
		block, ok := <-self.blocks
		if !ok {
			return
		}

		inHash, data, err := decodeDataBlock(block)
		if err != nil {
			logrus.Errorf("error decoding data block (%v)", err)
		}

		outHash := sha512.Sum512(data)
		if len(outHash) != len(inHash) {
			logrus.Errorf("hash length mismatch [%d != %d]", len(outHash), len(inHash))
		}
		for i := 0; i < len(outHash); i++ {
			if outHash[i] != inHash[i] {
				logrus.Errorf("hash mismatch at [#%d]", i)
				return
			}
		}

		block.unref()
	}
}
