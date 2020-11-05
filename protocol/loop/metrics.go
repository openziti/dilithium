package loop

import (
	"fmt"
	"github.com/openziti/dilithium/util"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"net"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

func init() {
	go signalHandler()
}

type Metrics struct {
	Addr   net.Addr
	Peer   net.Addr
	Prefix string
	close  chan struct{}

	RxBytes      []*util.Sample
	RxBytesAccum int64
	TxBytes      []*util.Sample
	TxBytesAccum int64
}

func NewMetrics(addr, peer net.Addr, ms int, prefix string) *Metrics {
	m := &Metrics{
		Addr:   addr,
		Peer:   peer,
		Prefix: prefix,
		close:  make(chan struct{}, 1),
	}
	registryLock.Lock()
	registry = append(registry, m)
	registryLock.Unlock()
	go m.snapshotter(ms)
	return m
}

func WriteAllSamples() {
	registryLock.Lock()
	defer registryLock.Unlock()

	for _, m := range registry {
		if err := m.writeSamples(); err != nil {
			logrus.Errorf("error writing samples (%v)", err)
		}
	}
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

func (self *Metrics) writeSamples() error {
	if err := os.MkdirAll(self.Prefix, os.ModePerm); err != nil {
		return err
	}
	name := fmt.Sprintf("%s_%s_", self.Addr, self.Peer)
	outPath, err := ioutil.TempDir(self.Prefix, name)
	if err != nil {
		return err
	}
	logrus.Infof("writing metrics to: %s", outPath)
	if err := util.WriteMetricsId("dilithiumLoop", outPath, nil); err != nil {
		return err
	}
	if err := util.WriteSamples("rxBytes", outPath, self.RxBytes); err != nil {
		return err
	}
	if err := util.WriteSamples("txBytes", outPath, self.TxBytes); err != nil {
		return err
	}
	return nil
}

var registry []*Metrics
var registryLock sync.Mutex

func signalHandler() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGUSR2)

	for {
		s := <-c
		if s == syscall.SIGUSR2 {
			WriteAllSamples()
		}
	}
}
