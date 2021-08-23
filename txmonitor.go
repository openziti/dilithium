package dilithium

import (
	"sync"
)

// TxMonitor is responsible for managing in-flight payloads, retransmitting payloads when their timeout expires.
//
type TxMonitor struct {
	lock         *sync.Mutex
	ready        *sync.Cond
	alg          TxAlgorithm
	transport    Transport
	rttAvg       []uint16
	retxMs       int
	retxCallback func()
}

func newTxMonitor(lock *sync.Mutex, alg TxAlgorithm, transport Transport) *TxMonitor {
	return &TxMonitor{
		lock:      lock,
		ready:     sync.NewCond(lock),
		alg:       alg,
		transport: transport,
	}
}

func (txm *TxMonitor) setRetxCallback(c func()) {
	txm.retxCallback = c
}
