package westworld2

import (
	"encoding/json"
	"fmt"
	"github.com/michaelquigley/dilithium/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

type metricsInstrument struct {
	prefix     string
	snapshotMs int
	lock       *sync.Mutex
	instances  []*metricsInstrumentInstance
}

func newMetricsInstrument(config map[string]interface{}) (Instrument, error) {
	mi := &metricsInstrument{
		snapshotMs: 1000,
		lock:       new(sync.Mutex),
	}
	if err := mi.configure(config); err != nil {
		return nil, err
	}
	go mi.signalHandler()
	return mi, nil
}

func (self *metricsInstrument) newInstance(peer *net.UDPAddr) InstrumentInstance {
	self.lock.Lock()
	defer self.lock.Unlock()
	mi := &metricsInstrumentInstance{
		peer:  peer,
		close: make(chan struct{}, 1),
	}
	self.instances = append(self.instances, mi)
	go mi.snapshotter(self.snapshotMs)
	return mi
}

func (self *metricsInstrument) configure(config map[string]interface{}) error {
	if v, found := config["prefix"]; found {
		if prefix, ok := v.(string); ok {
			self.prefix = prefix
			logrus.Infof("metrics data prefix set to [%s]", self.prefix)
		} else {
			return errors.New("invalid 'prefix' type")
		}
	}
	if v, found := config["snapshot_ms"]; found {
		if snapshotMs, ok := v.(int); ok {
			self.snapshotMs = snapshotMs
			logrus.Infof("snapshot interval set to [%d ms.]", self.snapshotMs)
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

func (self *metricsInstrument) writeAllSamples() error {
	self.lock.Lock()
	defer self.lock.Unlock()

	for _, mii := range self.instances {
		peerName := fmt.Sprintf("%s_", mii.peer.String())
		if err := os.MkdirAll(self.prefix, os.ModePerm); err != nil {
			return err
		}
		outPath, err := ioutil.TempDir(self.prefix, peerName)
		if err != nil {
			return err
		}
		logrus.Infof("writing metrics to: %s", outPath)

		if err := self.writeId(mii, outPath); err != nil {
			return err
		}
		if err := util.WriteSamples("txBytes", outPath, mii.txBytes); err != nil {
			return err
		}
		if err := util.WriteSamples("txMsgs", outPath, mii.txMsgs); err != nil {
			return err
		}
		if err := util.WriteSamples("retxBytes", outPath, mii.retxBytes); err != nil {
			return err
		}
		if err := util.WriteSamples("retxMsgs", outPath, mii.retxMs); err != nil {
			return err
		}
		if err := util.WriteSamples("rxBytes", outPath, mii.rxBytes); err != nil {
			return err
		}
		if err := util.WriteSamples("rxMsgs", outPath, mii.rxMsgs); err != nil {
			return err
		}
		if err := util.WriteSamples("txPortalCapacity", outPath, mii.txPortalCapacity); err != nil {
			return err
		}
		if err := util.WriteSamples("txPortalSz", outPath, mii.txPortalSz); err != nil {
			return err
		}
		if err := util.WriteSamples("txPortalRxSz", outPath, mii.txPortalRxSz); err != nil {
			return err
		}
		if err := util.WriteSamples("retxMs", outPath, mii.retxMs); err != nil {
			return err
		}
		if err := util.WriteSamples("dupAcks", outPath, mii.dupAcks); err != nil {
			return err
		}
		if err := util.WriteSamples("rxPortalSz", outPath, mii.rxPortalSz); err != nil {
			return err
		}
		if err := util.WriteSamples("dupRxBytes", outPath, mii.dupRxBytes); err != nil {
			return err
		}
		if err := util.WriteSamples("dupRxMsgs", outPath, mii.dupRxMsgs); err != nil {
			return err
		}
		if err := util.WriteSamples("allocations", outPath, mii.allocations); err != nil {
			return err
		}
		if err := util.WriteSamples("errors", outPath, mii.errors); err != nil {
			return err
		}
	}
	return nil
}

type MetricsId struct {
	Id     string            `json:"id"`
	Values map[string]string `json:"values"`
}

func (self *metricsInstrument) writeId(mii *metricsInstrumentInstance, outPath string) error {
	mid := &MetricsId{Id: "westworld2.1"}
	if mii.listenerAddr != nil {
		mid.Values = make(map[string]string)
		mid.Values["listener"] = mii.listenerAddr.String()
	}
	data, err := json.MarshalIndent(mid, "", "  ")
	if err != nil {
		return err
	}
	oF, err := os.OpenFile(filepath.Join(outPath, "metrics.id"), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return err
	}
	defer func() { _ = oF.Close() }()
	if _, err := oF.Write(data); err != nil {
		return err
	}
	return nil
}

type metricsInstrumentInstance struct {
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
func (self *metricsInstrumentInstance) listener(addr *net.UDPAddr) {
	self.listenerAddr = addr
}

func (self *metricsInstrumentInstance) connected(_ *net.UDPAddr) {
}

func (self *metricsInstrumentInstance) closed(_ *net.UDPAddr) {
	logrus.Warnf("closing snapshotter")
	close(self.close)
}

func (self *metricsInstrumentInstance) connectError(_ *net.UDPAddr, err error) {
	logrus.Errorf("connect error (%v)", err)
	atomic.AddInt64(&self.errorsAccum, 1)
}

/*
 * wire
 */
func (self *metricsInstrumentInstance) wireMessageTx(_ *net.UDPAddr, wm *wireMessage) {
	atomic.AddInt64(&self.txBytesAccum, int64(len(wm.data)))
	atomic.AddInt64(&self.txMsgsAccum, 1)
}

func (self *metricsInstrumentInstance) wireMessageRetx(_ *net.UDPAddr, wm *wireMessage) {
	atomic.AddInt64(&self.retxBytesAccum, int64(len(wm.data)))
	atomic.AddInt64(&self.retxMsgsAccum, 1)
}

func (self *metricsInstrumentInstance) wireMessageRx(_ *net.UDPAddr, wm *wireMessage) {
	atomic.AddInt64(&self.rxBytesAccum, int64(len(wm.data)))
	atomic.AddInt64(&self.rxMsgsAccum, 1)
}

func (self *metricsInstrumentInstance) unknownPeer(peer *net.UDPAddr) {
	logrus.Errorf("unknown peer (%v)", peer)
	atomic.AddInt64(&self.errorsAccum, 1)
}

func (self *metricsInstrumentInstance) readError(_ *net.UDPAddr, err error) {
	logrus.Errorf("read error (%v)", err)
	atomic.AddInt64(&self.errorsAccum, 1)
}

func (self *metricsInstrumentInstance) unexpectedMessageType(_ *net.UDPAddr, mt messageType) {
	logrus.Errorf("unexpected message type (%d)", mt)
	atomic.AddInt64(&self.errorsAccum, 1)
}

/*
 * txPortal
 */
func (self *metricsInstrumentInstance) txPortalCapacityChanged(_ *net.UDPAddr, capacity int) {
	atomic.StoreInt64(&self.txPortalCapacityVal, int64(capacity))
}

func (self *metricsInstrumentInstance) txPortalSzChanged(_ *net.UDPAddr, sz int) {
	atomic.StoreInt64(&self.txPortalSzVal, int64(sz))
}

func (self *metricsInstrumentInstance) txPortalRxSzChanged(_ *net.UDPAddr, sz int) {
	atomic.StoreInt64(&self.txPortalRxSzVal, int64(sz))
}

func (self *metricsInstrumentInstance) newRetxMs(_ *net.UDPAddr, ms int) {
	atomic.StoreInt64(&self.retxMsVal, int64(ms))
}

func (self *metricsInstrumentInstance) duplicateAck(_ *net.UDPAddr, _ int32) {
	atomic.AddInt64(&self.dupAcksAccum, 1)
}

/*
 * rxPortal
 */
func (self *metricsInstrumentInstance) rxPortalSzChanged(_ *net.UDPAddr, sz int) {
	atomic.StoreInt64(&self.rxPortalSzVal, int64(sz))
}

func (self *metricsInstrumentInstance) duplicateRx(_ *net.UDPAddr, wm *wireMessage) {
	atomic.AddInt64(&self.dupRxBytesAccum, int64(len(wm.data)))
	atomic.AddInt64(&self.dupRxMsgsAccum, 1)
}

/*
 * allocation
 */
func (self *metricsInstrumentInstance) allocate(_ string) {
	atomic.AddInt64(&self.allocationsAccum, 1)
}

/*
 * snapshotter
 */
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
