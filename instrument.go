package dilithium

import "github.com/pkg/errors"

type Instrument interface {
	NewInstance(id string) InstrumentInstance
}

type InstrumentInstance interface {
	// connection
	//Listener(adapter Adapter)
	//Hello(adapter Adapter)
	//Connected(adapter Adapter)
	//ConnectionError(adapter Adapter, err error)
	Closed(adapter Adapter)

	// wire
	WireMessageTx(wm *WireMessage)
	WireMessageRetx(wm *WireMessage)
	WireMessageRx(wm *WireMessage)
	// UnknownPeer(peer *net.UDPAddr)
	ReadError(err error)
	WriteError(err error)
	UnexpectedMessageType(mt messageType)

	// control
	TxAck(wm *WireMessage)
	RxAck(wm *WireMessage)
	TxKeepalive(wm *WireMessage)
	RxKeepalive(wm *WireMessage)

	// txPortal
	TxPortalCapacityChanged(capacity int)
	TxPortalSzChanged(capacity int)
	TxPortalRxSzChanged(sz int)
	NewRetxMs(retxMs int)
	NewRetxScale(retxScale float64)
	DuplicateAck(ack int32)

	// rxPortal
	RxPortalSzChanged(capacity int)
	DuplicateRx(wm *WireMessage)

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
