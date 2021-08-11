package dilithium

import (
	"github.com/emirpasic/gods/trees/btree"
	"sync"
)

// TxPortal manages the outgoing data transmitted by a communication instance. It is one half of a TxPortal->RxPortal
// communication pair. TxPortal is primarily concerned with optimizing the transmission rate over lossy Transport
// implementations, while ensuring reliability.
//
type TxPortal struct {
	lock         *sync.Mutex
	tree         *btree.Tree
	capacity     int
	ready        *sync.Cond
	txPortalSize int
	rxPortalSize int
	alg          TxAlgorithm
	monitor      *TxMonitor
	closer       *Closer
	transport    Transport
}

// TxAlgorithm is an abstraction of an extensible flow-control implementation, which can be plugged into a TxPortal
// instance.
//
type TxAlgorithm interface {
	Success()
	DuplicateAck()
	Retransmission()
}
