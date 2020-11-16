package westworld3

import (
	"fmt"
	"github.com/pkg/errors"
	"net"
	"strings"
	"sync"
)

type traceInstrument struct {
	wire     bool
	txPortal bool
	rxPortal bool
	error    bool
}

type traceInstrumentInstance struct {
	id   string
	peer *net.UDPAddr
	lock *sync.Mutex
	i    *traceInstrument
}

func NewTraceInstrument(config map[string]interface{}) (Instrument, error) {
	i := &traceInstrument{}
	if config != nil {
		if v, found := config["wire"]; found {
			if wire, ok := v.(bool); ok {
				i.wire = wire
			} else {
				return nil, errors.New("invalid 'wire' value")
			}
		}
		if v, found := config["tx_portal"]; found {
			if txPortal, ok := v.(bool); ok {
				i.txPortal = txPortal
			} else {
				return nil, errors.New("invalid 'tx_portal' value")
			}
		}
		if v, found := config["rx_portal"]; found {
			if rxPortal, ok := v.(bool); ok {
				i.rxPortal = rxPortal
			} else {
				return nil, errors.New("invalid 'rx_portal' value")
			}
		}
	}
	return i, nil
}

func (self *traceInstrument) NewInstance(id string, peer *net.UDPAddr) InstrumentInstance {
	return &traceInstrumentInstance{id, peer, new(sync.Mutex), self}
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
	if self.i.wire {
		decode, _ := self.decode(wm)
		self.lock.Lock()
		fmt.Println(fmt.Sprintf("&& %-24s %-8s #%-8d %s {%s} -> %s", self.id, "TX", wm.seq, wm.messageType(), wm.mt.FlagsString(), decode))
		self.lock.Unlock()
	}
}

func (self *traceInstrumentInstance) WireMessageRetx(peer *net.UDPAddr, wm *wireMessage) {
	if self.i.wire {
		decode, _ := self.decode(wm)
		self.lock.Lock()
		fmt.Println(fmt.Sprintf("&& %-24s %-8s #%-8d %s {%s} -> %s", self.id, "RETX", wm.seq, wm.messageType(), wm.mt.FlagsString(), decode))
		self.lock.Unlock()
	}
}

func (self *traceInstrumentInstance) WireMessageRx(peer *net.UDPAddr, wm *wireMessage) {
	if self.i.wire {
		decode, _ := self.decode(wm)
		self.lock.Lock()
		fmt.Println(fmt.Sprintf("&& %-24s %-8s #%-8d %s {%s} -> %s", self.id, "RX", wm.seq, wm.messageType(), wm.mt.FlagsString(), decode))
		self.lock.Unlock()
	}
}

func (self *traceInstrumentInstance) UnknownPeer(peer *net.UDPAddr) {
	if self.i.error {
		self.lock.Lock()
		fmt.Println(fmt.Sprintf("&& %-24s UNKNOWN PEER: %s", self.id, peer))
		self.lock.Unlock()
	}
}

func (self *traceInstrumentInstance) ReadError(peer *net.UDPAddr, err error) {
	if self.i.error {
		self.lock.Lock()
		fmt.Println(fmt.Sprintf("&& %-24s READ ERROR: %v", self.id, err))
		self.lock.Unlock()
	}
}

func (self *traceInstrumentInstance) UnexpectedMessageType(peer *net.UDPAddr, mt messageType) {
	if self.i.error {
		self.lock.Lock()
		fmt.Println(fmt.Sprintf("&& %-24s UNEXPECTED MESSAGE TYPE: %s", self.id, mt.String()))
		self.lock.Unlock()
	}
}

/*
 * txPortal
 */
func (self *traceInstrumentInstance) TxPortalCapacityChanged(peer *net.UDPAddr, capacity int) {
	if self.i.txPortal {
		self.lock.Lock()
		fmt.Println(fmt.Sprintf("!! %-24s TX PORTAL CAPACITY: %d", self.id, capacity))
		self.lock.Unlock()
	}
}

func (self *traceInstrumentInstance) TxPortalSzChanged(peer *net.UDPAddr, sz int) {
	if self.i.txPortal {
		self.lock.Lock()
		fmt.Println(fmt.Sprintf("!! %-24s TX PORTAL SZ: %d", self.id, sz))
		self.lock.Unlock()
	}
}

func (self *traceInstrumentInstance) TxPortalRxSzChanged(peer *net.UDPAddr, sz int) {
	if self.i.txPortal {
		self.lock.Lock()
		fmt.Println(fmt.Sprintf("!! %-24s TX PORTAL RX SZ: %d", self.id, sz))
		self.lock.Unlock()
	}
}

func (self *traceInstrumentInstance) NewRetxMs(peer *net.UDPAddr, retxMs int) {
	if self.i.txPortal {
		self.lock.Lock()
		fmt.Println(fmt.Sprintf("!! %-24s RETX MS: %d", self.id, retxMs))
		self.lock.Unlock()
	}
}

func (self *traceInstrumentInstance) DuplicateAck(peer *net.UDPAddr, seq int32) {
	if self.i.txPortal {
		self.lock.Lock()
		fmt.Println(fmt.Sprintf("!! %-24s DUPLICATE ACK: #%d", self.id, seq))
		self.lock.Unlock()
	}
}

/*
 * rxPortal
 */
func (self *traceInstrumentInstance) RxPortalSzChanged(peer *net.UDPAddr, sz int) {
	if self.i.rxPortal {
		self.lock.Lock()
		fmt.Println(fmt.Sprintf("!! %-24s RX PORTAL SZ: %d", self.id, sz))
		self.lock.Unlock()
	}
}

func (self *traceInstrumentInstance) DuplicateRx(peer *net.UDPAddr, wm *wireMessage) {
	if self.i.rxPortal {
		self.lock.Lock()
		fmt.Println(fmt.Sprintf("!! %-24s DUPLICATE RX: #%d", self.id, wm.seq))
		self.lock.Unlock()
	}
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
}

func (self *traceInstrumentInstance) decode(wm *wireMessage) (string, error) {
	out := ""
	switch wm.messageType() {
	case HELLO:
		h, acks, err := wm.asHello()
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("{v:%d, p:%d} |%s|", h.version, h.profile, self.decodeAcks(acks)), nil

	case ACK:
		a, rxPortalSz, _, err := wm.asAck()
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("|%s| %%%d", self.decodeAcks(a), rxPortalSz), nil

	case DATA:
		sz, err := wm.asDataSize()
		if err != nil {
			return "", err
		}
		return fmt.Sprintf(":%d", sz), nil

	default:
		return out, nil
	}
}

func (self *traceInstrumentInstance) decodeAcks(acks []ack) string {
	out := ""
	for _, ack := range acks {
		if ack.start == ack.end {
			out += fmt.Sprintf(" @%d", ack.start)
		} else {
			out += fmt.Sprintf(" @%d:%d", ack.start, ack.end)
		}
	}
	return strings.TrimSpace(out)
}
