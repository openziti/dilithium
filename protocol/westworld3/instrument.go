package westworld3

import "net"

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

	// txPortal
	TxPortalCapacityChanged(peer *net.UDPAddr, capacity int)
	TxPortalSzChanged(peer *net.UDPAddr, capacity int)
	TxPortalRxSzChanged(peer *net.UDPAddr, sz int)
	NewRetxMs(peer *net.UDPAddr, retxMs int)
	DuplicateAck(peer *net.UDPAddr, ack int32)

	// rxPortal
	RxPortalSzChanged(peer *net.UDPAddr, capacity int)
	DuplicateRx(peer *net.UDPAddr, wm *wireMessage)

	// allocation
	Allocate(id string)

	// instrument lifecycle
	Shutdown()
}