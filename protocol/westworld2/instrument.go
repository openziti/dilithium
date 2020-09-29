package westworld2

import (
	"github.com/pkg/errors"
	"net"
)

type Instrument interface {
	newInstance(peer *net.UDPAddr) InstrumentInstance
}

type InstrumentInstance interface {
	// connection
	connected(peer *net.UDPAddr)
	closed(peer *net.UDPAddr)
	connectError(peer *net.UDPAddr, err error)

	// wire
	wireMessageTx(peer *net.UDPAddr, wm *wireMessage)
	wireMessageRetx(peer *net.UDPAddr, wm *wireMessage)
	wireMessageRx(peer *net.UDPAddr, wm *wireMessage)
	unknownPeer(peer *net.UDPAddr)
	readError(peer *net.UDPAddr, err error)
	unexpectedMessageType(peer *net.UDPAddr, mt messageType)

	// txPortal
	txPortalCapacityChanged(peer *net.UDPAddr, capacity int)
	txPortalSzChanged(peer *net.UDPAddr, capacity int)
	txPortalRxPortalSzChanged(peer *net.UDPAddr, sz int)
	newRetxMs(peer *net.UDPAddr, retxMs int)
	duplicateAck(peer *net.UDPAddr, ack int32)

	// rxPortal
	rxPortalSzChanged(peer *net.UDPAddr, capacity int)
	duplicateRx(peer *net.UDPAddr, wm *wireMessage)

	// allocation
	allocate(ctx string)
}

func NewInstrument(name string, config map[string]interface{}) (i Instrument, err error) {
	switch name {
	case "logger":
		return newLoggerInstrument(), nil
	case "metrics":
		return newMetricsInstrument(config)
	case "nil":
		return nil, nil
	case "trace":
		return newTraceInstrument(), nil
	default:
		return nil, errors.Errorf("unknown instrument '%s'", name)
	}
}
