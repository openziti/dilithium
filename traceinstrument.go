package dilithium

import (
	"fmt"
	"github.com/openziti-incubator/cf"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"net"
	"strings"
	"sync"
)

type traceInstrument struct {
	config *traceInstrumentConfig
}

type traceInstrumentConfig struct {
	Wire     bool `cf:"wire"`
	Control  bool `cf:"control"`
	TxPortal bool `cf:"tx_portal"`
	RxPortal bool `cf:"rx_portal"`
	Error    bool `cf:"error"`
}

type traceInstrumentInstance struct {
	id   string
	lock sync.Mutex
	i    *traceInstrument
}

func NewTraceInstrument(config map[string]interface{}) (Instrument, error) {
	i := &traceInstrument{
		config: new(traceInstrumentConfig),
	}
	if err := cf.Bind(i.config, config, cf.DefaultOptions()); err != nil {
		return nil, errors.Wrap(err, "unable to load config")
	}
	logrus.Infof(cf.Dump(i.config, cf.DefaultOptions()))
	return i, nil
}

func (self *traceInstrument) NewInstance(id string) InstrumentInstance {
	return &traceInstrumentInstance{
		id: id,
		i:  self,
	}
}

/*
 * connection
 */

func (self *traceInstrumentInstance) Listener(addr *net.UDPAddr) {}

func (self *traceInstrumentInstance) Hello(peer *net.UDPAddr) {
}

func (self *traceInstrumentInstance) Connected(peer *net.UDPAddr) {
}

func (self *traceInstrumentInstance) ConnectionError(peer *net.UDPAddr, err error) {
}

func (self *traceInstrumentInstance) Closed(peer Adapter) {}

/*
 * wire
 */

func (self *traceInstrumentInstance) WireMessageTx(wm *WireMessage) {
	if self.i.config.Wire {
		decode, _ := self.decode(wm)
		self.lock.Lock()
		fmt.Println(fmt.Sprintf("&& %-24s %-8s #%-8d %s {%s} -> %s", self.id, "TX", wm.Seq, wm.messageType(), wm.Mt.FlagsString(), decode))
		self.lock.Unlock()
	}
}

func (self *traceInstrumentInstance) WireMessageRetx(wm *WireMessage) {
	if self.i.config.Wire {
		decode, _ := self.decode(wm)
		self.lock.Lock()
		fmt.Println(fmt.Sprintf("&& %-24s %-8s #%-8d %s {%s} -> %s", self.id, "RETX", wm.Seq, wm.messageType(), wm.Mt.FlagsString(), decode))
		self.lock.Unlock()
	}
}

func (self *traceInstrumentInstance) WireMessageRx(wm *WireMessage) {
	if self.i.config.Wire {
		decode, _ := self.decode(wm)
		self.lock.Lock()
		fmt.Println(fmt.Sprintf("&& %-24s %-8s #%-8d %s {%s} -> %s", self.id, "RX", wm.Seq, wm.messageType(), wm.Mt.FlagsString(), decode))
		self.lock.Unlock()
	}
}

func (self *traceInstrumentInstance) UnknownPeer(peer *net.UDPAddr) {
	if self.i.config.Error {
		self.lock.Lock()
		fmt.Println(fmt.Sprintf("&& %-24s UNKNOWN PEER: %s", self.id, peer))
		self.lock.Unlock()
	}
}

func (self *traceInstrumentInstance) ReadError(err error) {
	if self.i.config.Error {
		self.lock.Lock()
		fmt.Println(fmt.Sprintf("&& %-24s READ ERROR: %v", self.id, err))
		self.lock.Unlock()
	}
}

func (self *traceInstrumentInstance) WriteError(err error) {
	if self.i.config.Error {
		self.lock.Lock()
		fmt.Println(fmt.Sprintf("&& %-24s WRITE ERROR: %v", self.id, err))
		self.lock.Unlock()
	}
}

func (self *traceInstrumentInstance) UnexpectedMessageType(mt messageType) {
	if self.i.config.Error {
		self.lock.Lock()
		fmt.Println(fmt.Sprintf("&& %-24s UNEXPECTED MESSAGE TYPE: %s", self.id, mt.String()))
		self.lock.Unlock()
	}
}

/*
 * control
 */

func (self *traceInstrumentInstance) TxAck(*WireMessage) {
	if self.i.config.Control {
		self.lock.Lock()
		fmt.Println(fmt.Sprintf("!! %-24s TX ACK", self.id))
		self.lock.Unlock()
	}
}

func (self *traceInstrumentInstance) RxAck(*WireMessage) {
	if self.i.config.Control {
		self.lock.Lock()
		fmt.Println(fmt.Sprintf("!! %-24s RX ACK", self.id))
		self.lock.Unlock()
	}
}

func (self *traceInstrumentInstance) TxKeepalive(*WireMessage) {
	if self.i.config.Control {
		self.lock.Lock()
		fmt.Println(fmt.Sprintf("!! %-24s TX KEEPALIVE", self.id))
		self.lock.Unlock()
	}
}

func (self *traceInstrumentInstance) RxKeepalive(*WireMessage) {
	if self.i.config.Control {
		self.lock.Lock()
		fmt.Println(fmt.Sprintf("!! %-24s RX KEEPALIVE", self.id))
		self.lock.Unlock()
	}
}

/*
 * txPortal
 */

func (self *traceInstrumentInstance) TxPortalCapacityChanged(capacity int) {
	if self.i.config.TxPortal {
		self.lock.Lock()
		fmt.Println(fmt.Sprintf("!! %-24s TX PORTAL CAPACITY: %d", self.id, capacity))
		self.lock.Unlock()
	}
}

func (self *traceInstrumentInstance) TxPortalSzChanged(sz int) {
	if self.i.config.TxPortal {
		self.lock.Lock()
		fmt.Println(fmt.Sprintf("!! %-24s TX PORTAL SZ: %d", self.id, sz))
		self.lock.Unlock()
	}
}

func (self *traceInstrumentInstance) TxPortalRxSzChanged(sz int) {
	if self.i.config.TxPortal {
		self.lock.Lock()
		fmt.Println(fmt.Sprintf("!! %-24s TX PORTAL RX SZ: %d", self.id, sz))
		self.lock.Unlock()
	}
}

func (self *traceInstrumentInstance) NewRetxMs(retxMs int) {
	if self.i.config.TxPortal {
		self.lock.Lock()
		fmt.Println(fmt.Sprintf("!! %-24s RETX MS: %d", self.id, retxMs))
		self.lock.Unlock()
	}
}

func (self *traceInstrumentInstance) NewRetxScale(retxScale float64) {
	if self.i.config.TxPortal {
		self.lock.Lock()
		fmt.Println(fmt.Sprintf("!! %-24s RETX SCALE: %0.2f", self.id, retxScale))
		self.lock.Unlock()
	}
}

func (self *traceInstrumentInstance) DuplicateAck(seq int32) {
	if self.i.config.TxPortal {
		self.lock.Lock()
		fmt.Println(fmt.Sprintf("!! %-24s DUPLICATE ACK: #%d", self.id, seq))
		self.lock.Unlock()
	}
}

/*
 * rxPortal
 */
func (self *traceInstrumentInstance) RxPortalSzChanged(sz int) {
	if self.i.config.RxPortal {
		self.lock.Lock()
		fmt.Println(fmt.Sprintf("!! %-24s RX PORTAL SZ: %d", self.id, sz))
		self.lock.Unlock()
	}
}

func (self *traceInstrumentInstance) DuplicateRx(wm *WireMessage) {
	if self.i.config.RxPortal {
		self.lock.Lock()
		fmt.Println(fmt.Sprintf("!! %-24s DUPLICATE RX: #%d", self.id, wm.Seq))
		self.lock.Unlock()
	}
}

/*
 * allocation
 */

func (self *traceInstrumentInstance) Allocate(string) {
}

/*
 * instrument lifecycle
 */

func (self *traceInstrumentInstance) Shutdown() {
	self.lock.Lock()
	fmt.Println(fmt.Sprintf("@@ %-24s SHUTDOWN", self.id))
	self.lock.Unlock()
}

func (self *traceInstrumentInstance) decode(wm *WireMessage) (string, error) {
	out := ""
	switch wm.messageType() {
	case HELLO:
		h, acks, err := wm.asHello()
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("{v:%d} |%s|", h.version, self.decodeAcks(acks)), nil

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

func (self *traceInstrumentInstance) decodeAcks(acks []Ack) string {
	out := ""
	for _, ack := range acks {
		if ack.Start == ack.End {
			out += fmt.Sprintf(" @%d", ack.Start)
		} else {
			out += fmt.Sprintf(" @%d:%d", ack.Start, ack.End)
		}
	}
	return strings.TrimSpace(out)
}
