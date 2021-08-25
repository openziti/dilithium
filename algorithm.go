package dilithium

import "time"

// TxAlgorithm is an abstraction of an extensible flow-control implementation, which can be plugged into a TxPortal
// instance.
//
type TxAlgorithm interface {
	// Tx will block the caller until there is sufficient space on the wire to enqueue the message.
	//
	Tx(int)

	// Success will unblock Tx callers by freeing up space on the wire from a successfully received message.
	//
	Success(int)

	// DuplicateAck is a signal that a duplicate transmission was received by the receiver.
	//
	DuplicateAck()

	// Retransmission is a signal that an acknowledgement was not received from the receiver within the round-trip time
	// window, and the transmitter sent another copy of the message.
	//
	Retransmission(int)

	// ProbeRTT will return bool when the transmitter is due to probe round trip time. It will also record that true
	// response and will not return true again until the algorithm wants another RTT probe.
	//
	ProbeRTT() bool

	// UpdateRTT pushes a round-trip time probe result onto the algorithm, allowing it to adapt its RetxMs calculation
	// accordingly.
	//
	UpdateRTT(rttMs int)

	// RetxMs returns the current timeout value for retransmission events.
	//
	RetxMs() int

	// RxPortalSize returns the currently observed size of the receiver's buffer.
	//
	RxPortalSize() int

	// UpdateRxPortalSize updates the observed size of the receiver's buffer.
	//
	UpdateRxPortalSize(int)

	// Profile returns the requested tunables for this algorithm.
	//
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