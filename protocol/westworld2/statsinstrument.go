package westworld2

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"net"
	"sync/atomic"
	"time"
)

type statsInstrument struct {
	rxMessages     int32
	rxBytes        int64
	rxDupeMessages int32
	rxDupeBytes    int64
	rxDupeAcks     int32
	txMessages     int32
	txBytes        int64
	retxMessages   int32
	retxBytes      int64
	unknownPeers   int32
	readErrors     int32
	unexpectedMt   int32
	allocations    int32
}

func NewStatsInstrument() Instrument {
	si := &statsInstrument{}
	go si.dumper()
	return si
}

func (self *statsInstrument) connected(_ *net.UDPAddr) {
}

func (self *statsInstrument) wireMessageRx(_ *net.UDPAddr, wm *wireMessage) {
	atomic.AddInt32(&self.rxMessages, 1)
	atomic.AddInt64(&self.rxBytes, int64(len(wm.data)))
}

func (self *statsInstrument) wireMessageTx(_ *net.UDPAddr, wm *wireMessage) {
	atomic.AddInt32(&self.txMessages, 1)
	atomic.AddInt64(&self.txBytes, int64(len(wm.data)))
}

func (self *statsInstrument) wireMessageRetx(_ *net.UDPAddr, wm *wireMessage) {
	atomic.AddInt32(&self.retxMessages, 1)
	atomic.AddInt64(&self.retxBytes, int64(len(wm.data)))
}

func (self *statsInstrument) unknownPeer(_ *net.UDPAddr) {
	atomic.AddInt32(&self.unknownPeers, 1)
}

func (self *statsInstrument) readError(_ *net.UDPAddr, _ error) {
	atomic.AddInt32(&self.readErrors, 1)
}

func (self *statsInstrument) connectError(_ *net.UDPAddr, _ error) {
}

func (self *statsInstrument) unexpectedMessageType(_ *net.UDPAddr, _ messageType) {
	atomic.AddInt32(&self.unexpectedMt, 1)
}

func (self *statsInstrument) duplicateRx(_ *net.UDPAddr, wm *wireMessage) {
	atomic.AddInt32(&self.rxDupeMessages, 1)
	atomic.AddInt64(&self.rxDupeBytes, int64(len(wm.data)))
}

func (self *statsInstrument) duplicateAck(_ *net.UDPAddr, _ int32) {
	atomic.AddInt32(&self.rxDupeAcks, 1)
}

func (self *statsInstrument) allocate(_ string) {
	atomic.AddInt32(&self.allocations, 1)
}

func (self *statsInstrument) configure(data map[interface{}]interface{}) error {
	return nil
}

func (self *statsInstrument) dumper() {
	for {
		time.Sleep(5 * time.Second)
		out := "stats{\n"
		out += fmt.Sprintf("\t%-20s %d\n", "rxMessages", self.rxMessages)
		out += fmt.Sprintf("\t%-20s %d\n", "rxBytes", self.rxBytes)
		out += fmt.Sprintf("\t%-20s %d\n", "rxDupeMessages", self.rxDupeMessages)
		out += fmt.Sprintf("\t%-20s %d\n", "rxDupeBytes", self.rxDupeBytes)
		out += fmt.Sprintf("\t%-20s %d\n", "rxDupeAcks", self.rxDupeAcks)
		out += fmt.Sprintf("\t%-20s %d\n", "txMessages", self.txMessages)
		out += fmt.Sprintf("\t%-20s %d\n", "txBytes", self.txBytes)
		out += fmt.Sprintf("\t%-20s %d\n", "retxMessages", self.retxMessages)
		out += fmt.Sprintf("\t%-20s %d\n", "retxBytes", self.retxBytes)
		out += fmt.Sprintf("\t%-20s %d\n", "unknownPeers", self.unknownPeers)
		out += fmt.Sprintf("\t%-20s %d\n", "readErrors", self.readErrors)
		out += fmt.Sprintf("\t%-20s %d\n", "unexpectedMt", self.unexpectedMt)
		out += fmt.Sprintf("\t%-20s %d\n", "allocations", self.allocations)
		out += "}"
		logrus.Infof(out)
	}
}
