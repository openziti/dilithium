package loop

import (
	"fmt"
	"github.com/openziti/dilithium/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"net"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Metrics struct {
	Addr   net.Addr
	Peer   net.Addr
	Prefix string
	close  chan struct{}

	start        time.Time
	RxBytes      []*util.Sample
	RxBytesAccum int64
	TxBytes      []*util.Sample
	TxBytesAccum int64
}

func NewMetrics(addr, peer net.Addr, ms int, prefix string) (*Metrics, error) {
	m := &Metrics{
		Addr:   addr,
		Peer:   peer,
		Prefix: prefix,
		close:  make(chan struct{}, 1),
	}
	registryLock.Lock()
	registry = append(registry, m)
	registryLock.Unlock()
	cl, err := util.GetCtrlListener(prefix, "loop")
	if err != nil {
		return nil, errors.Wrap(err, "unable to get ctrl listener")
	}
	cl.AddCallback("write", func(string, net.Conn) (int64, error) {
		WriteAllSamples()
		return 0, nil
	})
	cl.Start()
	go m.snapshotter(ms)
	return m, nil
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

func (self *Metrics) Start() {
	self.start = time.Now()
}

func (self *Metrics) Rx(bytes int64) {
	atomic.AddInt64(&self.RxBytesAccum, bytes)
}

func (self *Metrics) Tx(bytes int64) {
	atomic.AddInt64(&self.TxBytesAccum, bytes)
}

func (self *Metrics) Summarize() {
	rxTotalBytes := int64(0)
	rxLastTimestamp := time.Time{}
	for _, sample := range self.RxBytes {
		if sample.V > 0 {
			rxTotalBytes += sample.V
			rxLastTimestamp = sample.Ts
		}
	}
	rxDurationSeconds := float64(rxLastTimestamp.Sub(self.start).Milliseconds() / 1000.0)
	rxBytesSec := int64(float64(rxTotalBytes) / rxDurationSeconds)
	if rxTotalBytes > 0 {
		logrus.Infof("Rx: %s in %0.2f sec = %s/sec", util.BytesToSize(rxTotalBytes), rxDurationSeconds, util.BytesToSize(rxBytesSec))
	}

	txTotalBytes := int64(0)
	txLastTimestamp := time.Time{}
	for _, sample := range self.TxBytes {
		if sample.V > 0 {
			txTotalBytes += sample.V
			txLastTimestamp = sample.Ts
		}
	}
	txDurationSeconds := float64(txLastTimestamp.Sub(self.start).Milliseconds() / 1000.0)
	txBytesSec := int64(float64(txTotalBytes) / txDurationSeconds)
	if txTotalBytes > 0 {
		logrus.Infof("Tx: %s in %0.2f sec = %s/sec", util.BytesToSize(txTotalBytes), txDurationSeconds, util.BytesToSize(txBytesSec))
	}
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
	name := strings.ReplaceAll(fmt.Sprintf("loop_%s_%s_", self.Addr, self.Peer), ":", "-")
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
