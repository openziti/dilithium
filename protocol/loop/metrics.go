package loop

import (
	"github.com/michaelquigley/dilithium/util"
	"github.com/sirupsen/logrus"
	"net"
	"sync/atomic"
	"time"
)

type Metrics struct {
	Addr   *net.TCPAddr
	Peer   *net.TCPAddr
	Prefix string
	close  chan struct{}

	RxBytes      []*util.Sample
	RxBytesAccum int64
	TxBytes      []*util.Sample
	TxBytesAccum int64
}

func NewMetrics(addr, peer *net.TCPAddr, ms int, prefix string) *Metrics {
	m := &Metrics{
		Addr:   addr,
		Peer:   peer,
		Prefix: prefix,
		close:  make(chan struct{}, 1),
	}
	go m.snapshotter(ms)
	return m
}

func (self *Metrics) Rx(bytes int64) {
	atomic.AddInt64(&self.RxBytesAccum, bytes)
}

func (self *Metrics) Tx(bytes int64) {
	atomic.AddInt64(&self.TxBytesAccum, bytes)
}

func (self *Metrics) Close() {
	close(self.close)
}

func (self *Metrics) snapshotter(ms int) {
	logrus.Infof("started")
	defer logrus.Infof("exited")
	for {
		time.Sleep(time.Duration(ms) * time.Millisecond)
		self.RxBytes = append(self.RxBytes, &util.Sample{Ts: time.Now(), V: atomic.SwapInt64(&self.RxBytesAccum, 0)})
		self.TxBytes = append(self.TxBytes, &util.Sample{Ts: time.Now(), V: atomic.SwapInt64(&self.TxBytesAccum, 0)})
		select {
		case <-self.close:
			return
		default:
			//
		}
	}
}