package westworld2

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"net"
	"sync"
	"time"
)

type traceInstrument struct {
	lock   *sync.Mutex
	buffer []string
}

func NewTraceInstrument() Instrument {
	ti := &traceInstrument{
		lock: new(sync.Mutex),
	}
	go ti.dumper()
	return ti
}

func (self *traceInstrument) connected(peer *net.UDPAddr) {
	self.append(fmt.Sprintf("#| %-64s [%s]", "CONNECTED", peer))
}

func (self *traceInstrument) wireMessageRx(peer *net.UDPAddr, wm *wireMessage) {
	decode := fmt.Sprintf("%-12s", "RX " + self.mt(wm.mt))
	if wm.seq != -1 {
		decode += fmt.Sprintf("%-8s", fmt.Sprintf("#%d", wm.seq))
	}
	if wm.ack != -1 {
		decode += fmt.Sprintf("%-8s", fmt.Sprintf("@%d", wm.ack))
	}
	if len(wm.data) > 0 {
		decode += fmt.Sprintf("[%d]", len(wm.data))
	}
	self.append(fmt.Sprintf("#| %-64s [%s]", decode, peer))
}

func (self *traceInstrument) wireMessageTx(peer *net.UDPAddr, wm *wireMessage) {
	decode := fmt.Sprintf("%-12s", "TX " + self.mt(wm.mt))
	if wm.seq != -1 {
		decode += fmt.Sprintf("%-8s", fmt.Sprintf("#%d", wm.seq))
	}
	if wm.ack != -1 {
		decode += fmt.Sprintf("%-8s", fmt.Sprintf("@%d", wm.ack))
	}
	if len(wm.data) > 0 {
		decode += fmt.Sprintf("[%d]", len(wm.data))
	}
	self.append(fmt.Sprintf("#| %-64s [%s]", decode, peer))
}

func (self *traceInstrument) wireMessageRetx(peer *net.UDPAddr, wm *wireMessage) {
	decode := fmt.Sprintf("%-12s", "RETX " + self.mt(wm.mt))
	if wm.seq != -1 {
		decode += fmt.Sprintf("%-8s", fmt.Sprintf("#%d", wm.seq))
	}
	if wm.ack != -1 {
		decode += fmt.Sprintf("%-8s", fmt.Sprintf("@%d", wm.ack))
	}
	if len(wm.data) > 0 {
		decode += fmt.Sprintf("[%d]", len(wm.data))
	}
	self.append(fmt.Sprintf("#| %-64s [%s]", decode, peer))
}

func (self *traceInstrument) unknownPeer(peer *net.UDPAddr) {
	logrus.Errorf("unknown peer [%s]", peer)
}

func (self *traceInstrument) readError(peer *net.UDPAddr, err error) {
	logrus.Errorf("read error, peer [%s] (%v)", peer, err)
}

func (self *traceInstrument) connectError(peer *net.UDPAddr, err error) {
	logrus.Errorf("connect failed, peer [%s] (%v)", peer, err)
}

func (self *traceInstrument) unexpectedMessageType(peer *net.UDPAddr, mt messageType) {
	logrus.Errorf("unexpected message type [%s], peer [%s]", mtString(mt), peer)
}

func (self *traceInstrument) duplicateRx(peer *net.UDPAddr, wm *wireMessage) {
	decode := fmt.Sprintf("%-12s", "DPRX " + self.mt(wm.mt))
	if wm.seq != -1 {
		decode += fmt.Sprintf("%-8s", fmt.Sprintf("#%d", wm.seq))
	}
	if wm.ack != -1 {
		decode += fmt.Sprintf("%-8s", fmt.Sprintf("@%d", wm.ack))
	}
	if len(wm.data) > 0 {
		decode += fmt.Sprintf("[%d]", len(wm.data))
	}
	self.append(fmt.Sprintf("#| %-64s [%s]", decode, peer))
}

func (self *traceInstrument) duplicateAck(peer *net.UDPAddr, ack int32) {
	decode := fmt.Sprintf("%-12s", "DPACK")
	decode += fmt.Sprintf("%-8s", fmt.Sprintf("@%d", ack))
	self.append(fmt.Sprintf("#| %-64s [%s]", decode, peer))
}

func (self *traceInstrument) allocate(ctx string) {
	decode := fmt.Sprintf("%-12s %s", "ALLOCATE", ctx)
	self.append(fmt.Sprintf("#| %-64s", decode))
}

func (self *traceInstrument) mt(mt messageType) string {
	switch mt {
	case HELLO:
		return "HELLO"
	case ACK:
		return "ACK"
	case DATA:
		return "DATA"
	case CLOSE:
		return "CLOSE"
	default:
		return "????"
	}
}

func (self *traceInstrument) append(msg string) {
	self.lock.Lock()
	defer self.lock.Unlock()
	self.buffer = append(self.buffer, msg)
}

func (self *traceInstrument) dumper() {
	for {
		time.Sleep(1 * time.Second)
		self.lock.Lock()
		for _, msg := range self.buffer {
			fmt.Println(msg)
		}
		self.buffer = nil
		self.lock.Unlock()
	}
}
