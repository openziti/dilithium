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
	configure(data map[interface{}]interface{}) error
}

func NewInstrument(name string, _ map[interface{}]interface{}) (i Instrument, err error) {
	switch name {
	case "logger":
		return NewLoggerInstrument(), nil
	case "nil":
		return nil, nil
	case "stats":
		return NewStatsInstrument(), nil
	case "trace":
		return NewTraceInstrument(), nil
	default:
		return nil, errors.Errorf("unknown instrument '%s'", name)
	}
}
