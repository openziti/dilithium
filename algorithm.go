package dilithium

import "time"

// TxAlgorithm is an abstraction of an extensible flow-control implementation, which can be plugged into a TxPortal
// instance.
//
type TxAlgorithm interface {
	Ready(int)
	Tx(int)
	Success(int)
	DuplicateAck()
	Retransmission(int)
	UpdateRTT(rttMs int)
	RetxMs() int
	RxPortalSize() int
	Profile() *TxProfile
}

// TxProfile defines all of the configurable values that are requested by a flow control algorithm.
//
type TxProfile struct {
	RetxBatchMs       int
	SendKeepalive     bool
	ConnectionTimeout time.Duration
	MaxTreeSize       int
}
