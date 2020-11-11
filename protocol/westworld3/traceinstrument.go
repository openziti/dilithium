package westworld3

import (
	"fmt"
	"net"
)

type traceInstrument struct{}

type traceInstrumentInstance struct {
	id    string
	peer  *net.UDPAddr
	queue chan string
}

func NewTraceInstrument() Instrument {
	return &traceInstrument{}
}

func (self *traceInstrument) NewInstance(id string, peer *net.UDPAddr) InstrumentInstance {
	tii := &traceInstrumentInstance{id, peer, make(chan string, 1024)}
	go tii.run()
	return tii
}

/*
 * connection
 */
func (self *traceInstrumentInstance) Listener(addr *net.UDPAddr) {
}

func (self *traceInstrumentInstance) Hello(peer *net.UDPAddr) {
}

func (self *traceInstrumentInstance) Connected(peer *net.UDPAddr) {
}

func (self *traceInstrumentInstance) ConnectionError(peer *net.UDPAddr, err error) {
}

func (self *traceInstrumentInstance) Closed(peer *net.UDPAddr) {
}

/*
 * wire
 */
func (self *traceInstrumentInstance) WireMessageTx(peer *net.UDPAddr, wm *wireMessage) {
	self.queue <- fmt.Sprintf("&& %-24s %-8s #%-8d %s {%s}", self.id, "TX", wm.seq, wm.messageType(), wm.mt.FlagsString())
}

func (self *traceInstrumentInstance) WireMessageRetx(peer *net.UDPAddr, wm *wireMessage) {
	self.queue <- fmt.Sprintf("&& %-24s %-8s #%-8d %s {%s}", self.id, "RETX", wm.seq, wm.messageType(), wm.mt.FlagsString())
}

func (self *traceInstrumentInstance) WireMessageRx(peer *net.UDPAddr, wm *wireMessage) {
	self.queue <- fmt.Sprintf("&& %-24s %-8s #%-8d %s {%s}", self.id, "RX", wm.seq, wm.messageType(), wm.mt.FlagsString())
}

func (self *traceInstrumentInstance) UnknownPeer(peer *net.UDPAddr) {
}

func (self *traceInstrumentInstance) ReadError(peer *net.UDPAddr, err error) {
}

func (self *traceInstrumentInstance) UnexpectedMessageType(peer *net.UDPAddr, mt messageType) {
}

/*
 * txPortal
 */
func (self *traceInstrumentInstance) TxPortalCapacityChanged(peer *net.UDPAddr, capacity int) {
}

func (self *traceInstrumentInstance) TxPortalSzChanged(peer *net.UDPAddr, sz int) {
}

func (self *traceInstrumentInstance) TxPortalRxSzChanged(peer *net.UDPAddr, sz int) {
}

func (self *traceInstrumentInstance) NewRetxMs(peer *net.UDPAddr, retxMs int) {
}

func (self *traceInstrumentInstance) DuplicateAck(peer *net.UDPAddr, seq int32) {
}

/*
 * rxPortal
 */
func (self *traceInstrumentInstance) RxPortalSzChanged(peer *net.UDPAddr, sz int) {
}

func (self *traceInstrumentInstance) DuplicateRx(peer *net.UDPAddr, wm *wireMessage) {
}

/*
 * allocation
 */
func (self *traceInstrumentInstance) Allocate(id string) {
}

/*
 * instrument lifecycle
 */
func (self *traceInstrumentInstance) Shutdown() {
	close(self.queue)
}

func (self *traceInstrumentInstance) run() {
	for {
		select {
		case l, ok := <-self.queue:
			if !ok {
				return
			}
			fmt.Println(l)
		}
	}
}
