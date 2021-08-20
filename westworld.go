package dilithium

// WestworldAlgorithm implements the latest iteration of "westworld"-style flow control.
//
type WestworldAlgorithm struct {
	pf       *WestworldProfile
	txPortal *TxPortal
}

func NewWestworldAlgorithm(pf *WestworldProfile, txPortal *TxPortal) TxAlgorithm {
	return &WestworldAlgorithm{pf, txPortal}
}

func (self *WestworldAlgorithm) Ready() {
}

func (self *WestworldAlgorithm) Tx(size int) {
}

func (self *WestworldAlgorithm) Success(size int) {
}

func (self *WestworldAlgorithm) DuplicateAck() {
}

func (self *WestworldAlgorithm) Retransmission(size int) {
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
