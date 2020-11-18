package westworld3

import (
	"net"
	"sync"
)

type metricsInstrument struct {
	path       string
	snapshotMs int
	lock       *sync.Mutex
	instances  []*metricsInstrumentInstance
}

func (self *metricsInstrument) NewInstance(id string, peer *net.UDPAddr) InstrumentInstance {
	return &metricsInstrumentInstance{}
}

type metricsInstrumentInstance struct{}

/*
 * connection
 */
func (self *metricsInstrumentInstance) Listener(*net.UDPAddr)               {}
func (self *metricsInstrumentInstance) Hello(*net.UDPAddr)                  {}
func (self *metricsInstrumentInstance) Connected(*net.UDPAddr)              {}
func (self *metricsInstrumentInstance) ConnectionError(*net.UDPAddr, error) {}
func (self *metricsInstrumentInstance) Closed(*net.UDPAddr)                 {}

/*
 * wire
 */
func (self *metricsInstrumentInstance) WireMessageTx(*net.UDPAddr, *wireMessage)        {}
func (self *metricsInstrumentInstance) WireMessageRetx(*net.UDPAddr, *wireMessage)      {}
func (self *metricsInstrumentInstance) WireMessageRx(*net.UDPAddr, *wireMessage)        {}
func (self *metricsInstrumentInstance) UnknownPeer(*net.UDPAddr)                        {}
func (self *metricsInstrumentInstance) ReadError(*net.UDPAddr, error)                   {}
func (self *metricsInstrumentInstance) UnexpectedMessageType(*net.UDPAddr, messageType) {}

/*
 * txPortal
 */
func (self *metricsInstrumentInstance) TxPortalCapacityChanged(*net.UDPAddr, int) {}
func (self *metricsInstrumentInstance) TxPortalSzChanged(*net.UDPAddr, int)       {}
func (self *metricsInstrumentInstance) TxPortalRxSzChanged(*net.UDPAddr, int)     {}
func (self *metricsInstrumentInstance) NewRetxMs(*net.UDPAddr, int)               {}
func (self *metricsInstrumentInstance) DuplicateAck(*net.UDPAddr, int32)          {}

/*
 * rxPortal
 */
func (self *metricsInstrumentInstance) RxPortalSzChanged(*net.UDPAddr, int)    {}
func (self *metricsInstrumentInstance) DuplicateRx(*net.UDPAddr, *wireMessage) {}

/*
 * allocation
 */
func (self *metricsInstrumentInstance) Allocate(string) {}

/*
 * instrument lifecycle
 */
func (self *metricsInstrumentInstance) Shutdown() {}