package dilithium

import "time"

// TxAlgorithm is an abstraction of an extensible flow-control implementation, which can be plugged into a TxPortal
// instance.
//
type TxAlgorithm interface {
	Tx(int)
	Success(int)
	DuplicateAck()
	Retransmission(int)

	// ProbeRTT will return bool when the transmitter is due to probe round trip time. It will also record that true
	// response and will not return true again until the algorithm wants another RTT probe.
	//
	ProbeRTT() bool

	UpdateRTT(rttMs int)
	RetxMs() int
	RxPortalSize() int
	Profile() *TxProfile
}

// TxProfile defines all of the configurable values that are requested by a flow control algorithm.
//
type TxProfile struct {
	MaxSegmentSize    int
	RetxBatchMs       int
	SendKeepalive     bool
	ConnectionTimeout time.Duration
	MaxTreeSize       int
}
