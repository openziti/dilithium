package loop

import (
	"github.com/michaelquigley/dilithium/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"net"
	"time"
)

type Sender struct {
	headerPool *Pool
	ds         *DataSet
	conn       net.Conn
	seq        util.Sequence
	ct         int
	rate       *transferReporter
	Done       chan struct{}
}

func NewSender(ds *DataSet, conn net.Conn, ct int) *Sender {
	return &Sender{
		headerPool: NewPool(headerSz + 1),
		ds:         ds,
		conn:       conn,
		ct:         ct,
		rate:       newTransferReporter("tx"),
		Done:       make(chan struct{}),
	}
}

func (self *Sender) Run() {
	logrus.Info("starting")
	defer logrus.Info("exiting")

	go self.rate.run()

	if err := self.sendStart(); err != nil {
		logrus.Errorf("error sending start (%v)", err)
		return
	}
	if err := self.sendData(); err != nil {
		logrus.Errorf("error sending data (%v)", err)
		return
	}
	if err := self.sendEnd(); err != nil {
		logrus.Errorf("error sending end (%v)", err)
	}

	close(self.rate.in)
	close(self.Done)
}

func (self *Sender) sendStart() error {
	h := &header{uint32(self.seq.Next()), START, 0, self.headerPool.get()}
	if err := writeHeader(h, self.conn); err != nil {
		return err
	}
	h.buffer.unref()
	return nil
}

func (self *Sender) sendData() error {
	count := 0
	for i := 0; i < self.ct; i++ {
		for _, block := range self.ds.blocks {
			//logrus.Infof("sending block #%d [uz: %d, sz: %d]", count, block.uz, block.sz)

			h := &header{uint32(self.seq.Next()), DATA, block.uz, self.headerPool.get()}
			if err := writeHeader(h, self.conn); err != nil {
				return err
			}
			h.buffer.unref()

			n, err := self.conn.Write(block.data)
			if err != nil {
				return err
			}
			if n != int(block.uz) {
				return errors.Errorf("short data write [%d != %d]", n, block.uz)
			}

			self.rate.in <- &transferReport{time.Now(), int64(n)}

			count++
		}
	}
	return nil
}

func (self *Sender) sendEnd() error {
	h := &header{uint32(self.seq.Next()), END, 0, self.headerPool.get()}
	if err := writeHeader(h, self.conn); err != nil {
		return err
	}
	h.buffer.unref()
	return nil
}
