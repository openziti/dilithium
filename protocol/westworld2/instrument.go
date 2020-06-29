package westworld2

import (
	"github.com/pkg/errors"
	"net"
)

type Instrument interface {
	connected(peer *net.UDPAddr)
	closed(peer *net.UDPAddr)
	wireMessageTx(peer *net.UDPAddr, wm *wireMessage)
	wireMessageRx(peer *net.UDPAddr, wm *wireMessage)
	wireMessageRetx(peer *net.UDPAddr, wm *wireMessage)
	portalCapacitySz(peer *net.UDPAddr, capacity int)
	unknownPeer(peer *net.UDPAddr)
	readError(peer *net.UDPAddr, err error)
	connectError(peer *net.UDPAddr, err error)
	unexpectedMessageType(peer *net.UDPAddr, mt messageType)
	duplicateRx(peer *net.UDPAddr, wm *wireMessage)
	duplicateAck(peer *net.UDPAddr, ack int32)
	newRextMs(peer *net.UDPAddr, rextMs int)
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
	case "stats":
		return newStatsInstrument(), nil
	case "trace":
		return newTraceInstrument(), nil
	default:
		return nil, errors.Errorf("unknown instrument '%s'", name)
	}
}
