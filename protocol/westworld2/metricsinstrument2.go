package westworld2

import (
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"net"
	"sync"
	"sync/atomic"
)

type metricsInstrument2 struct {
	prefix    string
	lock      *sync.Mutex
	instances []*metricsInstrumentInstance2
}

func newMetricsInstrument2(config map[string]interface{}) (Instrument, error) {
	mi := &metricsInstrument2{lock: new(sync.Mutex)}
	if err := mi.configure(config); err != nil {
		return nil, err
	}
	return mi, nil
}

func (self *metricsInstrument2) newInstance(peer *net.UDPAddr) InstrumentInstance {
	self.lock.Lock()
	defer self.lock.Unlock()
	mi := &metricsInstrumentInstance2{peer: peer}
	self.instances = append(self.instances, mi)
	return mi
}

func (self *metricsInstrument2) configure(config map[string]interface{}) error {
	if v, found := config["prefix"]; found {
		if prefix, ok := v.(string); ok {
			self.prefix = prefix
			logrus.Infof("metrics data prefix set to [%s]", self.prefix)
		} else {
			return errors.New("invalid 'prefix' type")
		}
	}
	return nil
}

type metricsInstrumentInstance2 struct {
	peer *net.UDPAddr

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

/*
 * connection
 */
func (self *metricsInstrumentInstance2) connected(_ *net.UDPAddr) {
}

func (self *metricsInstrumentInstance2) closed(_ *net.UDPAddr) {
}

func (self *metricsInstrumentInstance2) connectError(_ *net.UDPAddr, err error) {
	logrus.Errorf("connect error (%v)", err)
	atomic.AddInt64(&self.errorsAccum, 1)
}

/*
 * wire
 */
func (self *metricsInstrumentInstance2) wireMessageTx(_ *net.UDPAddr, wm *wireMessage) {
	atomic.AddInt64(&self.txBytesAccum, int64(len(wm.data)))
	atomic.AddInt64(&self.txMsgsAccum, 1)
}

func (self *metricsInstrumentInstance2) wireMessageRetx(_ *net.UDPAddr, wm *wireMessage) {
	atomic.AddInt64(&self.retxBytesAccum, int64(len(wm.data)))
	atomic.AddInt64(&self.retxMsgsAccum, 1)
}

func (self *metricsInstrumentInstance2) wireMessageRx(_ *net.UDPAddr, wm *wireMessage) {
	atomic.AddInt64(&self.rxBytesAccum, int64(len(wm.data)))
	atomic.AddInt64(&self.rxMsgsAccum, 1)
}

func (self *metricsInstrumentInstance2) unknownPeer(peer *net.UDPAddr) {
	logrus.Errorf("unknown peer (%v)", peer)
	atomic.AddInt64(&self.errorsAccum, 1)
}

func (self *metricsInstrumentInstance2) readError(_ *net.UDPAddr, err error) {
	logrus.Errorf("read error (%v)", err)
	atomic.AddInt64(&self.errorsAccum, 1)
}

func (self *metricsInstrumentInstance2) unexpectedMessageType(_ *net.UDPAddr, mt messageType) {
	logrus.Errorf("unexpected message type (%d)", mt)
	atomic.AddInt64(&self.errorsAccum, 1)
}

/*
 * txPortal
 */
func (self *metricsInstrumentInstance2) txPortalCapacityChanged(_ *net.UDPAddr, capacity int) {
	atomic.StoreInt64(&self.txPortalCapacityVal, int64(capacity))
}

func (self *metricsInstrumentInstance2) txPortalSzChanged(_ *net.UDPAddr, sz int) {
	atomic.StoreInt64(&self.txPortalSzVal, int64(sz))
}

func (self *metricsInstrumentInstance2) txPortalRxPortalSzChanged(_ *net.UDPAddr, sz int) {
	atomic.StoreInt64(&self.txPortalRxSzVal, int64(sz))
}

func (self *metricsInstrumentInstance2) newRetxMs(_ *net.UDPAddr, ms int) {
	atomic.StoreInt64(&self.retxMsVal, int64(ms))
}

func (self *metricsInstrumentInstance2) duplicateAck(_ *net.UDPAddr, _ int32) {
	atomic.AddInt64(&self.dupAcksAccum, 1)
}

/*
 * rxPortal
 */
func (self *metricsInstrumentInstance2) rxPortalSzChanged(_ *net.UDPAddr, sz int) {
	atomic.StoreInt64(&self.rxPortalSzVal, int64(sz))
}

func (self *metricsInstrumentInstance2) duplicateRx(_ *net.UDPAddr, wm *wireMessage) {
	atomic.AddInt64(&self.dupRxBytesAccum, int64(len(wm.data)))
	atomic.AddInt64(&self.dupRxMsgsAccum, 1)
}

/*
 * allocation
 */
func (self *metricsInstrumentInstance2) allocate(_ string) {
	atomic.AddInt64(&self.allocationsAccum, 1)
}