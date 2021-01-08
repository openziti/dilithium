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
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var localEnabled = false
var localEnabledOverridden = false

type metricsInstrument struct {
	lock      *sync.Mutex
	config    *metricsInstrumentConfig
	instances []*metricsInstrumentInstance
}

type metricsInstrumentConfig struct {
	Path       string `cf:"path"`
	SnapshotMs int    `cf:"snapshot_ms"`
	Enabled    bool   `cf:"enabled"`
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
	if localEnabledOverridden {
		i.config.Enabled = localEnabled
	}
	cl, err := util.GetCtrlListener(i.config.Path, "westworld3")
	if err != nil {
		return nil, errors.Wrap(err, "unable to get metrics ctrl listener")
	}
	cl.AddCallback("start", func(string) error {
		localEnabled = true
		localEnabledOverridden = true

		i.config.Enabled = true
		return nil
	})
	cl.AddCallback("stop", func(string) error {
		localEnabled = false
		localEnabledOverridden = true

		i.config.Enabled = false
		return nil
	})
	cl.AddCallback("write", func(string) error {
		err := i.writeAllSamples()
		if err != nil {
			logrus.Errorf("error writing samples (%v)", err)
		}
		return err
	})
	cl.AddCallback("clean", func(string) error {
		i.clean()
		return nil
	})
	cl.Start()
	logrus.Infof(cf.Dump("config", i.config))
	return i, nil
}

func (self *metricsInstrument) NewInstance(id string, peer *net.UDPAddr) InstrumentInstance {
	self.lock.Lock()
	defer self.lock.Unlock()
	ii := &metricsInstrumentInstance{id: id, peer: peer, config: self.config, close: make(chan struct{}, 1)}
	go ii.snapshotter(self.config.SnapshotMs)
	self.instances = append(self.instances, ii)
	return ii
}

func (self *metricsInstrument) writeAllSamples() error {
	self.lock.Lock()
	defer self.lock.Unlock()

	for _, ii := range self.instances {
		peerName := strings.ReplaceAll(fmt.Sprintf("%s_", ii.id), ":", "-")
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
		if err := util.WriteSamples("retx_msgs", outPath, ii.retxMsgs); err != nil {
			return err
		}
		if err := util.WriteSamples("rx_bytes", outPath, ii.rxBytes); err != nil {
			return err
		}
		if err := util.WriteSamples("rx_msgs", outPath, ii.rxMsgs); err != nil {
			return err
		}
		if err := util.WriteSamples("tx_ack_bytes", outPath, ii.txAckBytes); err != nil {
			return err
		}
		if err := util.WriteSamples("tx_ack_msgs", outPath, ii.txAckMsgs); err != nil {
			return err
		}
		if err := util.WriteSamples("rx_ack_bytes", outPath, ii.rxAckBytes); err != nil {
			return err
		}
		if err := util.WriteSamples("rx_ack_msgs", outPath, ii.rxAckMsgs); err != nil {
			return err
		}
		if err := util.WriteSamples("tx_keepalive_bytes", outPath, ii.txKeepaliveBytes); err != nil {
			return err
		}
		if err := util.WriteSamples("tx_keepalive_msgs", outPath, ii.txKeepaliveMsgs); err != nil {
			return err
		}
		if err := util.WriteSamples("rx_keepalive_bytes", outPath, ii.rxKeepaliveBytes); err != nil {
			return err
		}
		if err := util.WriteSamples("rx_keepalive_msgs", outPath, ii.rxKeepaliveMsgs); err != nil {
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

func (self *metricsInstrument) clean() {
	self.lock.Lock()
	defer self.lock.Unlock()

	idx := self.findClosed()
	for idx != -1 {
		logrus.Infof("removed metricsInstrumentInstance #%p", self.instances[idx])
		self.instances = append(self.instances[:idx], self.instances[idx+1:]...)
		idx = self.findClosed()
	}
}

func (self *metricsInstrument) findClosed() int {
	for i, ii := range self.instances {
		if ii.closed {
			return i
		}
	}
	return -1
}

type metricsInstrumentInstance struct {
	id           string
	peer         *net.UDPAddr
	listenerAddr *net.UDPAddr
	config       *metricsInstrumentConfig
	close        chan struct{}
	closed       bool

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

	txAckBytes            []*util.Sample
	txAckBytesAccum       int64
	txAckMsgs             []*util.Sample
	txAckMsgsAccum        int64
	rxAckBytes            []*util.Sample
	rxAckBytesAccum       int64
	rxAckMsgs             []*util.Sample
	rxAckMsgsAccum        int64
	txKeepaliveBytes      []*util.Sample
	txKeepaliveBytesAccum int64
	txKeepaliveMsgs       []*util.Sample
	txKeepaliveMsgsAccum  int64
	rxKeepaliveBytes      []*util.Sample
	rxKeepaliveBytesAccum int64
	rxKeepaliveMsgs       []*util.Sample
	rxKeepaliveMsgsAccum  int64

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
	if !self.closed {
		self.closed = true
		close(self.close)
	}
}

/*
 * wire
 */
func (self *metricsInstrumentInstance) WireMessageTx(_ *net.UDPAddr, wm *wireMessage) {
	if self.config.Enabled {
		atomic.AddInt64(&self.txBytesAccum, int64(wm.buffer.uz))
		atomic.AddInt64(&self.txMsgsAccum, 1)
	}
}

func (self *metricsInstrumentInstance) WireMessageRetx(_ *net.UDPAddr, wm *wireMessage) {
	if self.config.Enabled {
		atomic.AddInt64(&self.retxBytesAccum, int64(wm.buffer.uz))
		atomic.AddInt64(&self.retxMsgsAccum, 1)
	}
}

func (self *metricsInstrumentInstance) WireMessageRx(_ *net.UDPAddr, wm *wireMessage) {
	if self.config.Enabled {
		atomic.AddInt64(&self.rxBytesAccum, int64(wm.buffer.uz))
		atomic.AddInt64(&self.rxMsgsAccum, 1)
	}
}

func (self *metricsInstrumentInstance) UnknownPeer(peer *net.UDPAddr) {
	if self.config.Enabled {
		logrus.Errorf("unknown peer (%v)", peer)
		atomic.AddInt64(&self.errorsAccum, 1)
	}
}

func (self *metricsInstrumentInstance) ReadError(_ *net.UDPAddr, err error) {
	if self.config.Enabled {
		logrus.Errorf("read error (%v)", err)
		atomic.AddInt64(&self.errorsAccum, 1)
	}
}

func (self *metricsInstrumentInstance) UnexpectedMessageType(_ *net.UDPAddr, mt messageType) {
	if self.config.Enabled {
		logrus.Errorf("unexpected message type (%d)", mt)
		atomic.AddInt64(&self.errorsAccum, 1)
	}
}

/*
 * control
 */
func (self *metricsInstrumentInstance) TxAck(_ *net.UDPAddr, wm *wireMessage) {
	if self.config.Enabled {
		atomic.AddInt64(&self.txAckBytesAccum, int64(wm.buffer.uz))
		atomic.AddInt64(&self.txAckMsgsAccum, 1)
	}
}

func (self *metricsInstrumentInstance) RxAck(_ *net.UDPAddr, wm *wireMessage) {
	if self.config.Enabled {
		atomic.AddInt64(&self.rxAckBytesAccum, int64(wm.buffer.uz))
		atomic.AddInt64(&self.rxAckMsgsAccum, 1)
	}
}

func (self *metricsInstrumentInstance) TxKeepalive(_ *net.UDPAddr, wm *wireMessage) {
	if self.config.Enabled {
		atomic.AddInt64(&self.txKeepaliveBytesAccum, int64(wm.buffer.uz))
		atomic.AddInt64(&self.txKeepaliveMsgsAccum, 1)
	}
}

func (self *metricsInstrumentInstance) RxKeepalive(_ *net.UDPAddr, wm *wireMessage) {
	if self.config.Enabled {
		atomic.AddInt64(&self.rxKeepaliveBytesAccum, int64(wm.buffer.uz))
		atomic.AddInt64(&self.rxKeepaliveMsgsAccum, 1)
	}
}

/*
 * txPortal
 */
func (self *metricsInstrumentInstance) TxPortalCapacityChanged(_ *net.UDPAddr, capacity int) {
	if self.config.Enabled {
		atomic.StoreInt64(&self.txPortalCapacityVal, int64(capacity))
	}
}

func (self *metricsInstrumentInstance) TxPortalSzChanged(_ *net.UDPAddr, sz int) {
	if self.config.Enabled {
		atomic.StoreInt64(&self.txPortalSzVal, int64(sz))
	}
}

func (self *metricsInstrumentInstance) TxPortalRxSzChanged(_ *net.UDPAddr, sz int) {
	if self.config.Enabled {
		atomic.StoreInt64(&self.txPortalRxSzVal, int64(sz))
	}
}

func (self *metricsInstrumentInstance) NewRetxMs(_ *net.UDPAddr, ms int) {
	if self.config.Enabled {
		atomic.StoreInt64(&self.retxMsVal, int64(ms))
	}
}

func (self *metricsInstrumentInstance) NewRetxScale(_ *net.UDPAddr, retxMs float64) {
	if self.config.Enabled {
		atomic.StoreInt64(&self.retxScaleVal, int64(retxMs*1000.0))
	}
}

func (self *metricsInstrumentInstance) DuplicateAck(*net.UDPAddr, int32) {
	if self.config.Enabled {
		atomic.AddInt64(&self.dupAcksAccum, 1)
	}
}

/*
 * rxPortal
 */
func (self *metricsInstrumentInstance) RxPortalSzChanged(_ *net.UDPAddr, sz int) {
	if self.config.Enabled {
		atomic.StoreInt64(&self.rxPortalSzVal, int64(sz))
	}
}

func (self *metricsInstrumentInstance) DuplicateRx(_ *net.UDPAddr, wm *wireMessage) {
	if self.config.Enabled {
		atomic.AddInt64(&self.dupRxBytesAccum, int64(wm.buffer.uz))
		atomic.AddInt64(&self.dupRxMsgsAccum, 1)
	}
}

/*
 * allocation
 */
func (self *metricsInstrumentInstance) Allocate(string) {
	if self.config.Enabled {
		atomic.AddInt64(&self.allocationsAccum, 1)
	}
}

/*
 * instrument lifecycle
 */
func (self *metricsInstrumentInstance) Shutdown() {
	if !self.closed {
		self.closed = true
		close(self.close)
	}
}

func (self *metricsInstrumentInstance) snapshotter(ms int) {
	logrus.Infof("started")
	defer logrus.Infof("exited")
	for {
		time.Sleep(time.Duration(ms) * time.Millisecond)
		if self.config.Enabled {
			self.snapshot()
		}
		select {
		case <-self.close:
			self.snapshot()
			return
		default:
			//
		}
	}
}

func (self *metricsInstrumentInstance) snapshot() {
	now := time.Now()
	self.txBytes = append(self.txBytes, &util.Sample{Ts: now, V: atomic.SwapInt64(&self.txBytesAccum, 0)})
	self.txMsgs = append(self.txMsgs, &util.Sample{Ts: now, V: atomic.SwapInt64(&self.txMsgsAccum, 0)})
	self.retxBytes = append(self.retxBytes, &util.Sample{Ts: now, V: atomic.SwapInt64(&self.retxBytesAccum, 0)})
	self.retxMsgs = append(self.retxMsgs, &util.Sample{Ts: now, V: atomic.SwapInt64(&self.retxMsgsAccum, 0)})
	self.rxBytes = append(self.rxBytes, &util.Sample{Ts: now, V: atomic.SwapInt64(&self.rxBytesAccum, 0)})
	self.rxMsgs = append(self.rxMsgs, &util.Sample{Ts: now, V: atomic.SwapInt64(&self.rxMsgsAccum, 0)})
	self.txAckBytes = append(self.txAckBytes, &util.Sample{Ts: now, V: atomic.SwapInt64(&self.txAckBytesAccum, 0)})
	self.txAckMsgs = append(self.txAckMsgs, &util.Sample{Ts: now, V: atomic.SwapInt64(&self.txAckMsgsAccum, 0)})
	self.rxAckBytes = append(self.rxAckBytes, &util.Sample{Ts: now, V: atomic.SwapInt64(&self.rxAckBytesAccum, 0)})
	self.rxAckMsgs = append(self.rxAckMsgs, &util.Sample{Ts: now, V: atomic.SwapInt64(&self.rxAckMsgsAccum, 0)})
	self.txKeepaliveBytes = append(self.txKeepaliveBytes, &util.Sample{Ts: now, V: atomic.SwapInt64(&self.txKeepaliveBytesAccum, 0)})
	self.txKeepaliveMsgs = append(self.txKeepaliveMsgs, &util.Sample{Ts: now, V: atomic.SwapInt64(&self.txKeepaliveMsgsAccum, 0)})
	self.rxKeepaliveBytes = append(self.rxKeepaliveBytes, &util.Sample{Ts: now, V: atomic.SwapInt64(&self.rxKeepaliveBytesAccum, 0)})
	self.rxKeepaliveMsgs = append(self.rxKeepaliveMsgs, &util.Sample{Ts: now, V: atomic.SwapInt64(&self.rxKeepaliveMsgsAccum, 0)})
	self.txPortalCapacity = append(self.txPortalCapacity, &util.Sample{Ts: now, V: atomic.LoadInt64(&self.txPortalCapacityVal)})
	self.txPortalSz = append(self.txPortalSz, &util.Sample{Ts: now, V: atomic.LoadInt64(&self.txPortalSzVal)})
	self.txPortalRxSz = append(self.txPortalRxSz, &util.Sample{Ts: now, V: atomic.LoadInt64(&self.txPortalRxSzVal)})
	self.retxMs = append(self.retxMs, &util.Sample{Ts: now, V: atomic.LoadInt64(&self.retxMsVal)})
	self.retxScale = append(self.retxScale, &util.Sample{Ts: now, V: atomic.LoadInt64(&self.retxScaleVal)})
	self.dupAcks = append(self.dupAcks, &util.Sample{Ts: now, V: atomic.SwapInt64(&self.dupAcksAccum, 0)})
	self.rxPortalSz = append(self.rxPortalSz, &util.Sample{Ts: now, V: atomic.LoadInt64(&self.rxPortalSzVal)})
	self.dupRxBytes = append(self.dupRxBytes, &util.Sample{Ts: now, V: atomic.SwapInt64(&self.dupRxBytesAccum, 0)})
	self.dupRxMsgs = append(self.dupRxMsgs, &util.Sample{Ts: now, V: atomic.SwapInt64(&self.dupRxMsgsAccum, 0)})
	self.allocations = append(self.allocations, &util.Sample{Ts: now, V: atomic.SwapInt64(&self.allocationsAccum, 0)})
	self.errors = append(self.errors, &util.Sample{Ts: now, V: atomic.SwapInt64(&self.errorsAccum, 0)})
}
