package dilithium

import (
	"github.com/emirpasic/gods/trees/btree"
	"sync"
)

type TxPortal struct {
	lock     *sync.Mutex
	tree     *btree.Tree
	capacity int
	ready    *sync.Cond
}

type TxAlgorithm interface {
	Success()
	DuplicateAck()
	Retransmission()
}