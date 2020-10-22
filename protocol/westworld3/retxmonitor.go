package westworld3

import (
	"github.com/emirpasic/gods/trees/btree"
	"sync"
	"time"
)

type retxMonitor struct {
	waiting *btree.Tree
	ready   *sync.Cond
}

type retxSubject struct {
	deadline time.Time
	wm       *wireMessage
}

type waitlist interface {
	Add(*wireMessage, time.Time)
	Remove(*wireMessage)
	Next(retxSubject) (time.Time, bool)
	All() []*wireMessage
}
