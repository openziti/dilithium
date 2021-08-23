package dilithium

import (
	"sync"
)

// TxAlgorithm is an abstraction of an extensible flow-control implementation, which can be plugged into a TxPortal
// instance.
//
type TxAlgorithm interface {
	Ready()
	Tx(int)
	Success(int)
	DuplicateAck()
	Retransmission(int)
	UpdateRTT(rttMs int)
	RetxMs() int
	RetxBatchMs() int
}

// TxPortal manages the outgoing data transmitted by a communication instance. It is one half of a TxPortal->RxPortal
// communication pair. TxPortal is primarily concerned with optimizing the transmission rate over lossy Transport
// implementations, while ensuring reliability.
//
type TxPortal struct {
	lock      *sync.Mutex
	transport Transport
	alg       TxAlgorithm
	monitor   *TxMonitor
	closer    *Closer
	pool      *Pool
}

func newTxPortal(transport Transport, alg TxAlgorithm, closer *Closer, pool *Pool) *TxPortal {
	txp := &TxPortal{
		lock:      new(sync.Mutex),
		transport: transport,
		alg:       alg,
		closer:    closer,
	}
	txp.monitor = newTxMonitor(txp.lock, txp.alg, txp.transport)
	return txp
}
