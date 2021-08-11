package dilithium

import (
	"github.com/emirpasic/gods/trees/btree"
	"sync"
)

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

type TxAlgorithm interface {
	Success()
	DuplicateAck()
	Retransmission()
}
