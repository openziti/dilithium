package westworld2

import (
	"net"
)

type Instrument interface {
	connected(peer *net.UDPAddr)
	wireMessageTx(peer *net.UDPAddr, wm *wireMessage)
	wireMessageRx(peer *net.UDPAddr, wm *wireMessage)
	wireMessageRetx(peer *net.UDPAddr, wm *wireMessage)
	unknownPeer(peer *net.UDPAddr)
	readError(peer *net.UDPAddr, err error)
	connectError(peer *net.UDPAddr, err error)
	unexpectedMessageType(peer *net.UDPAddr, mt messageType)
	duplicateRx(peer *net.UDPAddr, wm *wireMessage)
	duplicateAck(peer *net.UDPAddr, ack int32)
	allocate(ctx string)
}
