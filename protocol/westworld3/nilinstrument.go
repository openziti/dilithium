package westworld3

import "net"

type nilInstrument struct{}

func NewNilInstrument() Instrument {
	return &nilInstrument{}
}

func (self *nilInstrument) NewInstance(_ string, _ *net.UDPAddr) InstrumentInstance {
	return &nilInstrumentInstance{}
}

type nilInstrumentInstance struct{}

/*
 * connection
 */
func (self *nilInstrumentInstance) Listener(*net.UDPAddr)               {}
func (self *nilInstrumentInstance) Hello(*net.UDPAddr)                  {}
func (self *nilInstrumentInstance) Connected(*net.UDPAddr)              {}
func (self *nilInstrumentInstance) ConnectionError(*net.UDPAddr, error) {}
func (self *nilInstrumentInstance) Closed(*net.UDPAddr)                 {}

/*
 * wire
 */
func (self *nilInstrumentInstance) WireMessageTx(*net.UDPAddr, *wireMessage)        {}
func (self *nilInstrumentInstance) WireMessageRetx(*net.UDPAddr, *wireMessage)      {}
func (self *nilInstrumentInstance) WireMessageRx(*net.UDPAddr, *wireMessage)        {}
func (self *nilInstrumentInstance) UnknownPeer(*net.UDPAddr)                        {}
func (self *nilInstrumentInstance) ReadError(*net.UDPAddr, error)                   {}
func (self *nilInstrumentInstance) UnexpectedMessageType(*net.UDPAddr, messageType) {}

/*
 * control
 */
func (self *nilInstrumentInstance) TxAck(*net.UDPAddr, *wireMessage)       {}
func (self *nilInstrumentInstance) RxAck(*net.UDPAddr, *wireMessage)       {}
func (self *nilInstrumentInstance) TxKeepalive(*net.UDPAddr, *wireMessage) {}
func (self *nilInstrumentInstance) RxKeepalive(*net.UDPAddr, *wireMessage) {}

/*
 * txPortal
 */
func (self *nilInstrumentInstance) TxPortalCapacityChanged(*net.UDPAddr, int) {}
func (self *nilInstrumentInstance) TxPortalSzChanged(*net.UDPAddr, int)       {}
func (self *nilInstrumentInstance) TxPortalRxSzChanged(*net.UDPAddr, int)     {}
func (self *nilInstrumentInstance) NewRetxMs(*net.UDPAddr, int)               {}
func (self *nilInstrumentInstance) NewRetxScale(*net.UDPAddr, float64)        {}
func (self *nilInstrumentInstance) DuplicateAck(*net.UDPAddr, int32)          {}

/*
 * rxPortal
 */
func (self *nilInstrumentInstance) RxPortalSzChanged(*net.UDPAddr, int)    {}
func (self *nilInstrumentInstance) DuplicateRx(*net.UDPAddr, *wireMessage) {}

/*
 * allocation
 */
func (self *nilInstrumentInstance) Allocate(string) {}

/*
 * instrument lifecycle
 */
func (self *nilInstrumentInstance) Shutdown() {}
