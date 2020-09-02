package loop

import (
	"github.com/michaelquigley/dilithium/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"net"
)

type sender struct {
	ds   *dataSet
	pool *pool
	conn net.Conn
	seq  util.Sequence
	ct   int
}

func newSender(ds *dataSet, pool *pool, conn net.Conn, ct int) *sender {
	return &sender{
		ds:   ds,
		pool: pool,
		conn: conn,
		ct:   ct,
	}
}

func (self *sender) run() {
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
}

func (self *sender) sendStart() error {
	h := &header{uint32(self.seq.Next()), START, 0, self.pool.get()}
	if err := writeHeader(h, self.conn); err != nil {
		return err
	}
	return nil
}

func (self *sender) sendData() error {
	for i := 0; i < self.ct; i++ {
		logrus.Infof("sending data, count #%d", i)
		for j, block := range self.ds.blocks {
			logrus.Infof("sending block, count #%d", j)
			n, err := self.conn.Write(block.buffer.data[:block.buffer.uz])
			if err != nil {
				return err
			}
			if n != int(block.buffer.uz) {
				return errors.Errorf("short data write [%d != %d]", n, block.buffer.uz)
			}
		}
	}
	return nil
}
