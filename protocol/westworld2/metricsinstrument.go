package westworld2

import (
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"net"
	"sync"
	"time"
)

type metricsInstrument struct {
	prefix           string
	lock             *sync.Mutex
	txBytes          samples
	retxBytes        samples
	rxBytes          samples
	txPortalSz       samples
	duplicateRxBytes samples
	duplicateAcks    samples
	rextMs           samples
	allocations      samples
}

type sample struct {
	ts time.Time
	v  int64
}

type samples []*sample

func (self samples) last() *sample {
	if len(self) > 0 {
		return self[len(self)-1]
	}
	return nil
}

func newMetricsInstrument(config map[string]interface{}) (Instrument, error) {
	mi := &metricsInstrument{
		lock: new(sync.Mutex),
	}
	if err := mi.configure(config); err != nil {
		return nil, err
	}
	return mi, nil
}

func (self *metricsInstrument) connected(_ *net.UDPAddr) {
}

func (self *metricsInstrument) closed(_ *net.UDPAddr) {
	logrus.Infof("writing metrics to prefix [%s]", self.prefix)
}

func (self *metricsInstrument) wireMessageRx(_ *net.UDPAddr, wm *wireMessage) {
	self.lock.Lock()
	self.rxBytes = append(self.rxBytes, &sample{time.Now(), int64(len(wm.data))})
	self.lock.Unlock()
}

func (self *metricsInstrument) wireMessageTx(_ *net.UDPAddr, wm *wireMessage) {
	self.lock.Lock()
	self.txBytes = append(self.txBytes, &sample{time.Now(), int64(len(wm.data))})
	self.lock.Unlock()
}

func (self *metricsInstrument) wireMessageRetx(_ *net.UDPAddr, wm *wireMessage) {
	self.lock.Lock()
	self.retxBytes = append(self.retxBytes, &sample{time.Now(), int64(len(wm.data))})
	self.lock.Unlock()
}

func (self *metricsInstrument) portalCapacitySz(_ *net.UDPAddr, capacity int) {
	self.lock.Lock()
	self.txPortalSz = append(self.txPortalSz, &sample{time.Now(), int64(capacity)})
	self.lock.Unlock()
}

func (self *metricsInstrument) unknownPeer(_ *net.UDPAddr) {
}

func (self *metricsInstrument) readError(_ *net.UDPAddr, _ error) {
}

func (self *metricsInstrument) connectError(_ *net.UDPAddr, _ error) {
}

func (self *metricsInstrument) unexpectedMessageType(_ *net.UDPAddr, _ messageType) {
}

func (self *metricsInstrument) duplicateRx(_ *net.UDPAddr, wm *wireMessage) {
	self.lock.Lock()
	self.duplicateRxBytes = append(self.duplicateRxBytes, &sample{time.Now(), int64(len(wm.data))})
	self.lock.Unlock()
}

func (self *metricsInstrument) duplicateAck(_ *net.UDPAddr, _ int32) {
	self.lock.Lock()
	self.duplicateAcks = append(self.duplicateAcks, &sample{time.Now(), 1})
	self.lock.Unlock()
}

func (self *metricsInstrument) newRextMs(_ *net.UDPAddr, rextMs int) {
	self.lock.Lock()
	self.rextMs = append(self.rextMs, &sample{time.Now(), int64(rextMs)})
	self.lock.Unlock()
}

func (self *metricsInstrument) allocate(_ string) {
	self.lock.Lock()
	self.allocations = append(self.allocations, &sample{time.Now(), 1})
	self.lock.Unlock()
}

func (self *metricsInstrument) configure(data map[string]interface{}) error {
	if v, found := data["prefix"]; found {
		if prefix, ok := v.(string); ok {
			self.prefix = prefix
			logrus.Infof("writing metrics data to prefix [%s]", self.prefix)
		} else {
			return errors.New("invalid 'prefix' type")
		}
	}
	return nil
}
