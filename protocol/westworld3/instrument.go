package westworld3

import "net"

type Instrument interface {
	newInstance(id string, peer *net.UDPAddr) InstrumentInstance
}

type InstrumentInstance interface {
	// allocation
	allocate(id string)
}