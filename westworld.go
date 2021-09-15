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
	rttAvg             []uint16
	retxMs             int

	wpf   *WestworldProfile
	pf    *TxProfile
	lock  *sync.Mutex
	ready *sync.Cond
}

func NewWestworldAlgorithm(pf *WestworldProfile) TxAlgorithm {
	wa := &WestworldAlgorithm{
		capacity:           pf.StartSize,
		txPortalSize:       0,
		rxPortalSize:       0,
		successCount:       0,
		successAccumulator: 0,
		dupAckCount:        0,
		retxCount:          0,
		lastRttProbe:       time.Time{},
		retxMs:             pf.RetxStartMs,

		wpf:  pf,
		pf:   pf.Txpf,
		lock: new(sync.Mutex),
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
	if wa.successCount == wa.wpf.SuccessThresh {
		wa.updateCapacity(wa.capacity + int(float64(wa.successAccumulator)*wa.wpf.SuccessScale))
		wa.successCount = 0
		wa.successAccumulator = 0
	}
	wa.ready.Broadcast()
}

func (wa *WestworldAlgorithm) DuplicateAck() {
	wa.dupAckCount++
	wa.successCount = 0
	if wa.dupAckCount >= wa.wpf.DupAckThresh {
		wa.updateCapacity(int(float64(wa.capacity) * wa.wpf.DupAckCapacityScale))
		wa.dupAckCount = 0
		wa.successAccumulator = int(float64(wa.successAccumulator) * wa.wpf.DupAckSuccessScale)
	}
}

func (wa *WestworldAlgorithm) Retransmission(_ int) {
	wa.retxCount++
	wa.successCount = 0
	if wa.retxCount >= wa.wpf.RetxThresh {
		wa.updateCapacity(int(float64(wa.capacity) * wa.wpf.RetxCapacityScale))
		wa.retxCount = 0
		wa.successAccumulator = int(float64(wa.successAccumulator) * wa.wpf.RetxSuccessScale)
	}
}

func (wa *WestworldAlgorithm) ProbeRTT() bool {
	if time.Since(wa.lastRttProbe).Milliseconds() >= int64(wa.wpf.RttProbeMs) {
		wa.lastRttProbe = time.Now()
		return true
	}
	return false
}

func (wa *WestworldAlgorithm) UpdateRTT(rttMs int) {
	wa.rttAvg = append(wa.rttAvg, uint16(rttMs))
	if len(wa.rttAvg) > wa.wpf.RttProbeAvg {
		wa.rttAvg = wa.rttAvg[1:]
	}
	if len(wa.rttAvg) == wa.wpf.RttProbeAvg {
		accum := 0
		for _, rttMs := range wa.rttAvg {
			accum += int(rttMs)
		}
		accum /= len(wa.rttAvg)
		wa.retxMs = accum + wa.wpf.RetxAddMs
	}
}

func (wa *WestworldAlgorithm) RetxMs() int {
	return wa.retxMs
}

func (wa *WestworldAlgorithm) RxPortalSize() int {
	return wa.rxPortalSize
}

func (wa *WestworldAlgorithm) UpdateRxPortalSize(rxPortalSize int) {
	wa.rxPortalSize = rxPortalSize
}

func (wa *WestworldAlgorithm) RxPortalPacing(newSize, oldSize int) bool {
	return false
}

func (wa *WestworldAlgorithm) Profile() *TxProfile {
	return wa.pf
}

func (wa *WestworldAlgorithm) availableCapacity(segmentSize int) bool {
	txPortalCapacity := float64(wa.capacity - int(float64(wa.rxPortalSize)*wa.wpf.RxSizePressureScale) - (wa.txPortalSize + segmentSize))
	rxPortalCapacity := float64(wa.capacity - (wa.rxPortalSize + segmentSize))
	return math.Min(txPortalCapacity, rxPortalCapacity) > 0
}

func (wa *WestworldAlgorithm) updateCapacity(capacity int) {
	wa.capacity = capacity
	if wa.capacity < wa.wpf.MinSize {
		wa.capacity = wa.wpf.MinSize
	}
	if wa.capacity > wa.wpf.MaxSize {
		wa.capacity = wa.wpf.MaxSize
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
	RetxStartMs         int
	RetxAddMs           int
	RetxThresh          int
	RetxCapacityScale   float64
	RetxSuccessScale    float64
	RxSizePressureScale float64
	RttProbeMs          int
	RttProbeAvg         int
	Txpf                *TxProfile
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
		RetxStartMs:         200,
		RetxAddMs:           0,
		RetxThresh:          64,
		RetxCapacityScale:   0.75,
		RetxSuccessScale:    0.825,
		RxSizePressureScale: 2.8911,
		RttProbeMs:          50,
		RttProbeAvg:         8,
		Txpf:                DefaultTxProfile(),
	}
}

func (wp *WestworldProfile) Create() (TxAlgorithm, error) {
	return NewWestworldAlgorithm(wp), nil
}
