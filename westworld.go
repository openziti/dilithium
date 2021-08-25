package dilithium

import (
	"math"
	"sync"
	"time"
)

// WestworldAlgorithm implements the latest iteration of "westworld"-style flow control.
//
type WestworldAlgorithm struct {
	capacity           int
	txPortalSize       int
	rxPortalSize       int
	successCount       int
	successAccumulator int
	dupAckCount        int
	retxCount          int
	lastRttProbe       time.Time

	pf       *WestworldProfile
	txPortal *TxPortal
	lock     *sync.Mutex
	ready    *sync.Cond
}

func NewWestworldAlgorithm(pf *WestworldProfile, txPortal *TxPortal) TxAlgorithm {
	wa := &WestworldAlgorithm{
		capacity:           pf.StartSize,
		txPortalSize:       0,
		rxPortalSize:       0,
		successCount:       0,
		successAccumulator: 0,
		dupAckCount:        0,
		retxCount:          0,
		lastRttProbe:       time.Time{},

		pf:       pf,
		txPortal: txPortal,
		lock:     new(sync.Mutex),
	}
	wa.ready = sync.NewCond(wa.lock)
	return wa
}

func (wa *WestworldAlgorithm) Tx(segmentSize int) {
	for !wa.availableCapacity(segmentSize) {
		wa.ready.Wait()
	}
	wa.txPortalSize += segmentSize
}

func (wa *WestworldAlgorithm) Success(segmentSize int) {
	wa.txPortalSize -= segmentSize
	wa.successCount++
	if wa.successCount == wa.pf.SuccessThresh {
		wa.updateCapacity(wa.capacity + int(float64(wa.successAccumulator)*wa.pf.SuccessScale))
		wa.successCount = 0
		wa.successAccumulator = 0
	}
	wa.ready.Broadcast()
}

func (wa *WestworldAlgorithm) DuplicateAck() {
}

func (wa *WestworldAlgorithm) Retransmission(segmentSize int) {
}

func (wa *WestworldAlgorithm) ProbeRTT() bool {
	return false
}

func (wa *WestworldAlgorithm) UpdateRTT(rttMs int) {
}

func (wa *WestworldAlgorithm) RetxMs() int {
	return 200
}

func (wa *WestworldAlgorithm) RxPortalSize() int {
	return 0
}

func (wa *WestworldAlgorithm) UpdateRxPortalSize(int) {
}

func (wa *WestworldAlgorithm) Profile() *TxProfile {
	return nil
}

func (wa *WestworldAlgorithm) availableCapacity(segmentSize int) bool {
	txPortalCapacity := float64(wa.capacity - int(float64(wa.rxPortalSize)*wa.pf.RxSizePressureScale) - (wa.txPortalSize + segmentSize))
	rxPortalCapacity := float64(wa.capacity - (wa.rxPortalSize + segmentSize))
	return math.Min(txPortalCapacity, rxPortalCapacity) > 0
}

func (wa *WestworldAlgorithm) updateCapacity(capacity int) {
	wa.capacity = capacity
	if wa.capacity < wa.pf.MinSize {
		wa.capacity = wa.pf.MinSize
	}
	if wa.capacity > wa.pf.MaxSize {
		wa.capacity = wa.pf.MaxSize
	}
}

type WestworldProfile struct {
	StartSize           int
	MinSize             int
	MaxSize             int
	SuccessThresh       int
	SuccessScale        float64
	DupAckThresh        int
	DupAckCapacityScale float64
	DupAckSuccessScale  float64
	RetxThresh          int
	RetxCapacityScale   float64
	RetxSuccessScale    float64
	RxSizePressureScale float64
}

func NewBaselineWestworldProfile() *WestworldProfile {
	return &WestworldProfile{
		StartSize:           96 * 1024,
		MinSize:             16 * 1024,
		MaxSize:             4 * 1024 * 1024,
		SuccessThresh:       224,
		SuccessScale:        1.0,
		DupAckThresh:        64,
		DupAckCapacityScale: 0.9,
		DupAckSuccessScale:  0.75,
		RetxThresh:          64,
		RetxCapacityScale:   0.75,
		RetxSuccessScale:    0.825,
		RxSizePressureScale: 2.8911,
	}
}
