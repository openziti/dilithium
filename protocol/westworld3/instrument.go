package westworld3

import (
	"github.com/pkg/errors"
	"net"
)

type Instrument interface {
	NewInstance(id string, peer *net.UDPAddr) InstrumentInstance
}

type InstrumentInstance interface {
	// connection
	Listener(addr *net.UDPAddr)
	Hello(peer *net.UDPAddr)
	Connected(peer *net.UDPAddr)
	ConnectionError(peer *net.UDPAddr, err error)
	Closed(peer *net.UDPAddr)

	// wire
	WireMessageTx(peer *net.UDPAddr, wm *wireMessage)
	WireMessageRetx(peer *net.UDPAddr, wm *wireMessage)
	WireMessageRx(peer *net.UDPAddr, wm *wireMessage)
	UnknownPeer(peer *net.UDPAddr)
	ReadError(peer *net.UDPAddr, err error)
	UnexpectedMessageType(peer *net.UDPAddr, mt messageType)

	// control
	TxAck(peer *net.UDPAddr, wm *wireMessage)
	RxAck(peer *net.UDPAddr, wm *wireMessage)
	TxKeepalive(peer *net.UDPAddr, wm *wireMessage)
	RxKeepalive(peer *net.UDPAddr, wm *wireMessage)

	// txPortal
	TxPortalCapacityChanged(peer *net.UDPAddr, capacity int)
	TxPortalSzChanged(peer *net.UDPAddr, capacity int)
	TxPortalRxSzChanged(peer *net.UDPAddr, sz int)
	NewRetxMs(peer *net.UDPAddr, retxMs int)
	NewRetxScale(peer *net.UDPAddr, retxScale float64)
	DuplicateAck(peer *net.UDPAddr, ack int32)

	// rxPortal
	RxPortalSzChanged(peer *net.UDPAddr, capacity int)
	DuplicateRx(peer *net.UDPAddr, wm *wireMessage)

	// allocation
	Allocate(id string)

	// instrument lifecycle
	Shutdown()
}

func NewInstrument(name string, config map[string]interface{}) (i Instrument, err error) {
	switch name {
	case "metrics":
		return NewMetricsInstrument(config)
	case "nil":
		return NewNilInstrument(), nil
	case "trace":
		return NewTraceInstrument(config)
	default:
		return nil, errors.Errorf("unknown instrument '%s'", name)
	}
}
