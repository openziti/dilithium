package westworld3

import (
	"fmt"
	"github.com/openziti/dilithium/cf"
	"github.com/openziti/dilithium/util"
	"github.com/pkg/errors"
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

type metricsInstrument struct {
	lock      *sync.Mutex
	config    *metricsInstrumentConfig
	instances []*metricsInstrumentInstance
}

type metricsInstrumentConfig struct {
	Path       string `cf:"path"`
	SnapshotMs int    `cf:"snapshot_ms"`
}

func NewMetricsInstrument(config map[string]interface{}) (Instrument, error) {
	i := &metricsInstrument{
		lock: new(sync.Mutex),
		config: &metricsInstrumentConfig{
			SnapshotMs: 1000,
		},
	}
	if err := cf.Load(config, i.config); err != nil {
		return nil, errors.Wrap(err, "unable to load config")
	}
	go i.signalHandler()
	logrus.Infof(cf.Dump("config", i.config))
	return i, nil
}

func (self *metricsInstrument) NewInstance(id string, peer *net.UDPAddr) InstrumentInstance {
	self.lock.Lock()
	defer self.lock.Unlock()
	ii := &metricsInstrumentInstance{id: id, peer: peer}
	go ii.snapshotter(self.config.SnapshotMs)
	self.instances = append(self.instances, ii)
	return ii
}

func (self *metricsInstrument) writeAllSamples() error {
	self.lock.Lock()
	defer self.lock.Unlock()

	for _, ii := range self.instances {
		peerName := fmt.Sprintf("%s_", ii.id)
		if err := os.MkdirAll(self.config.Path, os.ModePerm); err != nil {
			return err
		}
		outPath, err := ioutil.TempDir(self.config.Path, peerName)
		if err != nil {
			return err
		}
		logrus.Infof("writing metrics to: %s", outPath)

		var values map[string]string
		if ii.listenerAddr != nil {
			values = make(map[string]string)
			values["listener"] = ii.listenerAddr.String()
		}
		if err := util.WriteMetricsId(fmt.Sprintf("westworld3.%d", protocolVersion), outPath, values); err != nil {
			return err
		}
		if err := util.WriteSamples("tx_bytes", outPath, ii.txBytes); err != nil {
			return err
		}
		if err := util.WriteSamples("tx_msgs", outPath, ii.txMsgs); err != nil {
			return err
		}
		if err := util.WriteSamples("retx_bytes", outPath, ii.retxBytes); err != nil {
			return err
		}
		if err := util.WriteSamples("retx_msgs", outPath, ii.retxMs); err != nil {
			return err
		}
		if err := util.WriteSamples("rx_bytes", outPath, ii.rxBytes); err != nil {
			return err
		}
		if err := util.WriteSamples("rx_msgs", outPath, ii.rxMsgs); err != nil {
			return err
		}
		if err := util.WriteSamples("tx_portal_capacity", outPath, ii.txPortalCapacity); err != nil {
			return err
		}
		if err := util.WriteSamples("tx_portal_sz", outPath, ii.txPortalSz); err != nil {
			return err
		}
		if err := util.WriteSamples("tx_portal_rx_sz", outPath, ii.txPortalRxSz); err != nil {
			return err
		}
		if err := util.WriteSamples("retx_ms", outPath, ii.retxMs); err != nil {
			return err
		}
		if err := util.WriteSamples("retx_scale", outPath, ii.retxScale); err != nil {
			return err
		}
		if err := util.WriteSamples("dup_acks", outPath, ii.dupAcks); err != nil {
			return err
		}
		if err := util.WriteSamples("rx_portal_sz", outPath, ii.rxPortalSz); err != nil {
			return err
		}
		if err := util.WriteSamples("dup_rx_bytes", outPath, ii.dupRxBytes); err != nil {
			return err
		}
		if err := util.WriteSamples("dup_rx_msgs", outPath, ii.dupRxMsgs); err != nil {
			return err
		}
		if err := util.WriteSamples("allocations", outPath, ii.allocations); err != nil {
			return err
		}
		if err := util.WriteSamples("errors", outPath, ii.errors); err != nil {
			return err
		}
	}
	return nil
}

func (self *metricsInstrument) signalHandler() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGUSR2)

	for {
		s := <-c
		if s == syscall.SIGUSR2 {
			if err := self.writeAllSamples(); err != nil {
				logrus.Errorf("error writing all samples (%v)", err)
			}
		}
	}
}

type metricsInstrumentInstance struct {
	id           string
	peer         *net.UDPAddr
	listenerAddr *net.UDPAddr
	close        chan struct{}

	txBytes        []*util.Sample
	txBytesAccum   int64
	txMsgs         []*util.Sample
	txMsgsAccum    int64
	retxBytes      []*util.Sample
	retxBytesAccum int64
	retxMsgs       []*util.Sample
	retxMsgsAccum  int64
	rxBytes        []*util.Sample
	rxBytesAccum   int64
	rxMsgs         []*util.Sample
	rxMsgsAccum    int64

	txPortalCapacity    []*util.Sample
	txPortalCapacityVal int64
	txPortalSz          []*util.Sample
	txPortalSzVal       int64
	txPortalRxSz        []*util.Sample
	txPortalRxSzVal     int64
	retxMs              []*util.Sample
	retxMsVal           int64
	retxScale           []*util.Sample
	retxScaleVal        int64
	dupAcks             []*util.Sample
	dupAcksAccum        int64

	rxPortalSz      []*util.Sample
	rxPortalSzVal   int64
	dupRxBytes      []*util.Sample
	dupRxBytesAccum int64
	dupRxMsgs       []*util.Sample
	dupRxMsgsAccum  int64

	allocations      []*util.Sample
	allocationsAccum int64
	errors           []*util.Sample
	errorsAccum      int64
}

/*
 * connection
 */
func (self *metricsInstrumentInstance) Listener(addr *net.UDPAddr) {
	self.listenerAddr = addr
}

func (self *metricsInstrumentInstance) Hello(*net.UDPAddr)                  {}
func (self *metricsInstrumentInstance) Connected(*net.UDPAddr)              {}
func (self *metricsInstrumentInstance) ConnectionError(*net.UDPAddr, error) {}

func (self *metricsInstrumentInstance) Closed(*net.UDPAddr) {
	logrus.Infof("closing snapshotter")
	close(self.close)
}

/*
 * wire
 */
func (self *metricsInstrumentInstance) WireMessageTx(_ *net.UDPAddr, wm *wireMessage) {
	atomic.AddInt64(&self.txBytesAccum, int64(wm.buffer.uz))
	atomic.AddInt64(&self.txMsgsAccum, 1)
}

func (self *metricsInstrumentInstance) WireMessageRetx(_ *net.UDPAddr, wm *wireMessage) {
	atomic.AddInt64(&self.retxBytesAccum, int64(wm.buffer.uz))
	atomic.AddInt64(&self.retxMsgsAccum, 1)
}

func (self *metricsInstrumentInstance) WireMessageRx(_ *net.UDPAddr, wm *wireMessage) {
	atomic.AddInt64(&self.rxBytesAccum, int64(wm.buffer.uz))
	atomic.AddInt64(&self.rxMsgsAccum, 1)
}

func (self *metricsInstrumentInstance) UnknownPeer(peer *net.UDPAddr) {
	logrus.Errorf("unknown peer (%v)", peer)
	atomic.AddInt64(&self.errorsAccum, 1)
}

func (self *metricsInstrumentInstance) ReadError(_ *net.UDPAddr, err error) {
	logrus.Errorf("read error (%v)", err)
	atomic.AddInt64(&self.errorsAccum, 1)
}

func (self *metricsInstrumentInstance) UnexpectedMessageType(_ *net.UDPAddr, mt messageType) {
	logrus.Errorf("unexpected message type (%d)", mt)
	atomic.AddInt64(&self.errorsAccum, 1)
}

/*
 * txPortal
 */
func (self *metricsInstrumentInstance) TxPortalCapacityChanged(_ *net.UDPAddr, capacity int) {
	atomic.StoreInt64(&self.txPortalCapacityVal, int64(capacity))
}

func (self *metricsInstrumentInstance) TxPortalSzChanged(_ *net.UDPAddr, sz int) {
	atomic.StoreInt64(&self.txPortalSzVal, int64(sz))
}

func (self *metricsInstrumentInstance) TxPortalRxSzChanged(_ *net.UDPAddr, sz int) {
	atomic.StoreInt64(&self.txPortalRxSzVal, int64(sz))
}

func (self *metricsInstrumentInstance) NewRetxMs(_ *net.UDPAddr, ms int) {
	atomic.StoreInt64(&self.retxMsVal, int64(ms))
}

func (self *metricsInstrumentInstance) NewRetxScale(_ *net.UDPAddr, retxMs float64) {
	atomic.StoreInt64(&self.retxScaleVal, int64(retxMs*1000.0))
}

func (self *metricsInstrumentInstance) DuplicateAck(*net.UDPAddr, int32) {
	atomic.AddInt64(&self.dupAcksAccum, 1)
}

/*
 * rxPortal
 */
func (self *metricsInstrumentInstance) RxPortalSzChanged(_ *net.UDPAddr, sz int) {
	atomic.StoreInt64(&self.rxPortalSzVal, int64(sz))
}

func (self *metricsInstrumentInstance) DuplicateRx(_ *net.UDPAddr, wm *wireMessage) {
	atomic.AddInt64(&self.dupRxBytesAccum, int64(wm.buffer.uz))
	atomic.AddInt64(&self.dupRxMsgsAccum, 1)
}

/*
 * allocation
 */
func (self *metricsInstrumentInstance) Allocate(string) {
	atomic.AddInt64(&self.allocationsAccum, 1)
}

/*
 * instrument lifecycle
 */
func (self *metricsInstrumentInstance) Shutdown() {
	close(self.close)
}

func (self *metricsInstrumentInstance) snapshotter(ms int) {
	logrus.Infof("started")
	defer logrus.Infof("exited")
	for {
		time.Sleep(time.Duration(ms) * time.Millisecond)
		self.txBytes = append(self.txBytes, &util.Sample{Ts: time.Now(), V: atomic.SwapInt64(&self.txBytesAccum, 0)})
		self.txMsgs = append(self.txMsgs, &util.Sample{Ts: time.Now(), V: atomic.SwapInt64(&self.txMsgsAccum, 0)})
		self.retxBytes = append(self.retxBytes, &util.Sample{Ts: time.Now(), V: atomic.SwapInt64(&self.retxBytesAccum, 0)})
		self.retxMsgs = append(self.retxMsgs, &util.Sample{Ts: time.Now(), V: atomic.SwapInt64(&self.retxMsgsAccum, 0)})
		self.rxBytes = append(self.rxBytes, &util.Sample{Ts: time.Now(), V: atomic.SwapInt64(&self.rxBytesAccum, 0)})
		self.rxMsgs = append(self.rxMsgs, &util.Sample{Ts: time.Now(), V: atomic.SwapInt64(&self.rxMsgsAccum, 0)})
		self.txPortalCapacity = append(self.txPortalCapacity, &util.Sample{Ts: time.Now(), V: atomic.LoadInt64(&self.txPortalCapacityVal)})
		self.txPortalSz = append(self.txPortalSz, &util.Sample{Ts: time.Now(), V: atomic.LoadInt64(&self.txPortalSzVal)})
		self.txPortalRxSz = append(self.txPortalRxSz, &util.Sample{Ts: time.Now(), V: atomic.LoadInt64(&self.txPortalRxSzVal)})
		self.retxMs = append(self.retxMs, &util.Sample{Ts: time.Now(), V: atomic.LoadInt64(&self.retxMsVal)})
		self.retxScale = append(self.retxScale, &util.Sample{Ts: time.Now(), V: atomic.LoadInt64(&self.retxScaleVal)})
		self.dupAcks = append(self.dupAcks, &util.Sample{Ts: time.Now(), V: atomic.SwapInt64(&self.dupAcksAccum, 0)})
		self.rxPortalSz = append(self.rxPortalSz, &util.Sample{Ts: time.Now(), V: atomic.LoadInt64(&self.rxPortalSzVal)})
		self.dupRxBytes = append(self.dupRxBytes, &util.Sample{Ts: time.Now(), V: atomic.SwapInt64(&self.dupRxBytesAccum, 0)})
		self.dupRxMsgs = append(self.dupRxMsgs, &util.Sample{Ts: time.Now(), V: atomic.SwapInt64(&self.dupRxMsgsAccum, 0)})
		self.allocations = append(self.allocations, &util.Sample{Ts: time.Now(), V: atomic.SwapInt64(&self.allocationsAccum, 0)})
		self.errors = append(self.errors, &util.Sample{Ts: time.Now(), V: atomic.SwapInt64(&self.errorsAccum, 0)})
		select {
		case <-self.close:
			return
		default:
			//
		}
	}
}
