package westworld3

import "net"

type standbyInstrument struct{}

func NewStandbyInstrument() Instrument {
	return &standbyInstrument{}
}

func (self *standbyInstrument) NewInstance(_ string, _ *net.UDPAddr) InstrumentInstance {
	return &standbyInstrumentInstance{}
}

type standbyInstrumentInstance struct{}

/*
 * connection
 */
func (self *standbyInstrumentInstance) Listener(*net.UDPAddr)               {}
func (self *standbyInstrumentInstance) Hello(*net.UDPAddr)                  {}
func (self *standbyInstrumentInstance) Connected(*net.UDPAddr)              {}
func (self *standbyInstrumentInstance) ConnectionError(*net.UDPAddr, error) {}
func (self *standbyInstrumentInstance) Closed(*net.UDPAddr)                 {}

/*
 * wire
 */
func (self *standbyInstrumentInstance) WireMessageTx(*net.UDPAddr, *wireMessage)        {}
func (self *standbyInstrumentInstance) WireMessageRetx(*net.UDPAddr, *wireMessage)      {}
func (self *standbyInstrumentInstance) WireMessageRx(*net.UDPAddr, *wireMessage)        {}
func (self *standbyInstrumentInstance) UnknownPeer(*net.UDPAddr)                        {}
func (self *standbyInstrumentInstance) ReadError(*net.UDPAddr, error)                   {}
func (self *standbyInstrumentInstance) UnexpectedMessageType(*net.UDPAddr, messageType) {}

/*
 * control
 */
func (self *standbyInstrumentInstance) TxAck(*net.UDPAddr, *wireMessage)       {}
func (self *standbyInstrumentInstance) RxAck(*net.UDPAddr, *wireMessage)       {}
func (self *standbyInstrumentInstance) TxKeepalive(*net.UDPAddr, *wireMessage) {}
func (self *standbyInstrumentInstance) RxKeepalive(*net.UDPAddr, *wireMessage) {}

/*
 * txPortal
 */
func (self *standbyInstrumentInstance) TxPortalCapacityChanged(*net.UDPAddr, int) {}
func (self *standbyInstrumentInstance) TxPortalSzChanged(*net.UDPAddr, int)       {}
func (self *standbyInstrumentInstance) TxPortalRxSzChanged(*net.UDPAddr, int)     {}
func (self *standbyInstrumentInstance) NewRetxMs(*net.UDPAddr, int)               {}
func (self *standbyInstrumentInstance) NewRetxScale(*net.UDPAddr, float64)        {}
func (self *standbyInstrumentInstance) DuplicateAck(*net.UDPAddr, int32)          {}

/*
 * rxPortal
 */
func (self *standbyInstrumentInstance) RxPortalSzChanged(*net.UDPAddr, int)    {}
func (self *standbyInstrumentInstance) DuplicateRx(*net.UDPAddr, *wireMessage) {}

/*
 * allocation
 */
func (self *standbyInstrumentInstance) Allocate(string) {}

/*
 * instrument lifecycle
 */
func (self *standbyInstrumentInstance) Shutdown() {}
