package westworld3

import (
	"sync"
	"time"
)

type retxMonitor struct {
	waitlist waitlist
	ready    *sync.Cond
}

type waitlist interface {
	Add(*wireMessage, time.Time)
	Remove(*wireMessage)
	Next() (*wireMessage, time.Time, bool)
}

