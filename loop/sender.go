package loop

import (
	"github.com/michaelquigley/dilithium/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"net"
)

type Sender struct {
	ds   *DataSet
	pool *Pool
	conn net.Conn
	seq  util.Sequence
	ct   int
	Done chan struct{}
}

func NewSender(ds *DataSet, pool *Pool, conn net.Conn, ct int) *Sender {
	return &Sender{
		ds:   ds,
		pool: pool,
		conn: conn,
		ct:   ct,
		Done: make(chan struct{}),
	}
}

func (self *Sender) Run() {
	logrus.Info("starting")
	defer logrus.Info("exiting")

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
	close(self.Done)
}

func (self *Sender) sendStart() error {
	h := &header{uint32(self.seq.Next()), START, 0, self.pool.get()}
	if err := writeHeader(h, self.conn); err != nil {
		return err
	}
	return nil
}

func (self *Sender) sendData() error {
	count := 0
	for i := 0; i < self.ct; i++ {
		for _, block := range self.ds.blocks {
			logrus.Infof("sending block #%d", count)
			h := &header{uint32(self.seq.Next()), DATA, block.buffer.uz, self.pool.get()}
			if err := writeHeader(h, self.conn); err != nil {
				return err
			}
			n, err := self.conn.Write(block.buffer.data[:block.buffer.uz])
			if err != nil {
				return err
			}
			if n != int(block.buffer.uz) {
				return errors.Errorf("short data write [%d != %d]", n, block.buffer.uz)
			}
			count++
		}
	}
	return nil
}

func (self *Sender) sendEnd() error {
	h := &header{uint32(self.seq.Next()), END, 0, self.pool.get()}
	if err := writeHeader(h, self.conn); err != nil {
		return err
	}
	return nil
}
