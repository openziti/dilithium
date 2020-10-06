package westworld2

import (
	"fmt"
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
		outpath, err := ioutil.TempDir(self.prefix, peerName)
		if err != nil {
			return err
		}
		logrus.Infof("writing metrics to: %s", outpath)

		if err := self.writeSamples("txBytes", outpath, mii.txBytes); err != nil {
			return err
		}
		if err := self.writeSamples("txMsgs", outpath, mii.txMsgs); err != nil {
			return err
		}
		if err := self.writeSamples("retxBytes", outpath, mii.retxBytes); err != nil {
			return err
		}
		if err := self.writeSamples("retxMsgs", outpath, mii.retxMs); err != nil {
			return err
		}
		if err := self.writeSamples("rxBytes", outpath, mii.rxBytes); err != nil {
			return err
		}
		if err := self.writeSamples("rxMsgs", outpath, mii.rxMsgs); err != nil {
			return err
		}
		if err := self.writeSamples("txPortalCapacity", outpath, mii.txPortalCapacity); err != nil {
			return err
		}
		if err := self.writeSamples("txPortalSz", outpath, mii.txPortalSz); err != nil {
			return err
		}
		if err := self.writeSamples("txPortalRxSz", outpath, mii.txPortalRxSz); err != nil {
			return err
		}
		if err := self.writeSamples("retxMs", outpath, mii.retxMs); err != nil {
			return err
		}
		if err := self.writeSamples("dupAcks", outpath, mii.dupAcks); err != nil {
			return err
		}
		if err := self.writeSamples("rxPortalSz", outpath, mii.rxPortalSz); err != nil {
			return err
		}
		if err := self.writeSamples("dupRxBytes", outpath, mii.dupRxBytes); err != nil {
			return err
		}
		if err := self.writeSamples("dupRxMsgs", outpath, mii.dupRxMsgs); err != nil {
			return err
		}
		if err := self.writeSamples("allocations", outpath, mii.allocations); err != nil {
			return err
		}
		if err := self.writeSamples("errors", outpath, mii.errors); err != nil {
			return err
		}
	}
	return nil
}

func (self *metricsInstrument) writeSamples(name, outPath string, samples []*sample) error {
	path := filepath.Join(outPath, fmt.Sprintf("%s.csv", name))
	oF, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return err
	}
	defer func() {
		_ = oF.Close()
	}()
	for _, sample := range samples {
		line := fmt.Sprintf("%d,%d\n", sample.ts.UnixNano(), sample.v)
		n, err := oF.Write([]byte(line))
		if err != nil {
			return err
		}
		if n != len(line) {
			return errors.New("short write")
		}
	}
	logrus.Infof("wrote [%d] samples to [%s]", len(samples), path)
	return nil
}

type metricsInstrumentInstance struct {
	peer         *net.UDPAddr
	listenerAddr *net.UDPAddr
	close        chan struct{}

	txBytes        []*sample
	txBytesAccum   int64
	txMsgs         []*sample
	txMsgsAccum    int64
	retxBytes      []*sample
	retxBytesAccum int64
	retxMsgs       []*sample
	retxMsgsAccum  int64
	rxBytes        []*sample
	rxBytesAccum   int64
	rxMsgs         []*sample
	rxMsgsAccum    int64

	txPortalCapacity    []*sample
	txPortalCapacityVal int64
	txPortalSz          []*sample
	txPortalSzVal       int64
	txPortalRxSz        []*sample
	txPortalRxSzVal     int64
	retxMs              []*sample
	retxMsVal           int64
	dupAcks             []*sample
	dupAcksAccum        int64

	rxPortalSz      []*sample
	rxPortalSzVal   int64
	dupRxBytes      []*sample
	dupRxBytesAccum int64
	dupRxMsgs       []*sample
	dupRxMsgsAccum  int64

	allocations      []*sample
	allocationsAccum int64
	errors           []*sample
	errorsAccum      int64
}

type sample struct {
	ts time.Time
	v  int64
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
		self.txBytes = append(self.txBytes, &sample{time.Now(), atomic.SwapInt64(&self.txBytesAccum, 0)})
		self.txMsgs = append(self.txMsgs, &sample{time.Now(), atomic.SwapInt64(&self.txMsgsAccum, 0)})
		self.retxBytes = append(self.retxBytes, &sample{time.Now(), atomic.SwapInt64(&self.retxBytesAccum, 0)})
		self.retxMsgs = append(self.retxMsgs, &sample{time.Now(), atomic.SwapInt64(&self.retxMsgsAccum, 0)})
		self.rxBytes = append(self.rxBytes, &sample{time.Now(), atomic.SwapInt64(&self.rxBytesAccum, 0)})
		self.rxMsgs = append(self.rxMsgs, &sample{time.Now(), atomic.SwapInt64(&self.rxMsgsAccum, 0)})
		self.txPortalCapacity = append(self.txPortalCapacity, &sample{time.Now(), atomic.LoadInt64(&self.txPortalCapacityVal)})
		self.txPortalSz = append(self.txPortalSz, &sample{time.Now(), atomic.LoadInt64(&self.txPortalSzVal)})
		self.txPortalRxSz = append(self.txPortalRxSz, &sample{time.Now(), atomic.LoadInt64(&self.txPortalRxSzVal)})
		self.retxMs = append(self.retxMs, &sample{time.Now(), atomic.LoadInt64(&self.retxMsVal)})
		self.dupAcks = append(self.dupAcks, &sample{time.Now(), atomic.SwapInt64(&self.dupAcksAccum, 0)})
		self.rxPortalSz = append(self.rxPortalSz, &sample{time.Now(), atomic.LoadInt64(&self.rxPortalSzVal)})
		self.dupRxBytes = append(self.dupRxBytes, &sample{time.Now(), atomic.SwapInt64(&self.dupRxBytesAccum, 0)})
		self.dupRxMsgs = append(self.dupRxMsgs, &sample{time.Now(), atomic.SwapInt64(&self.dupRxMsgsAccum, 0)})
		self.allocations = append(self.allocations, &sample{time.Now(), atomic.SwapInt64(&self.allocationsAccum, 0)})
		self.errors = append(self.errors, &sample{time.Now(), atomic.SwapInt64(&self.errorsAccum, 0)})
		select {
		case <-self.close:
			return
		default:
			//
		}
	}
}
