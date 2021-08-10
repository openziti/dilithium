package dilithium

func NewWestAlgorithm(pf *WestProfile, txPortal *TxPortal) TxAlgorithm {
	return &WestAlgorithm{pf, txPortal}
}

type WestAlgorithm struct {
	pf       *WestProfile
	txPortal *TxPortal
}

func (self *WestAlgorithm) Success() {
}

func (self *WestAlgorithm) DuplicateAck() {
}

func (self *WestAlgorithm) Retransmission() {
}

type WestProfile struct {
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

func NewBaselineWestProfile() *WestProfile {
	return &WestProfile{
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
