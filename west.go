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
	TxPortalStartSize int
	TxPortalMinSize   int
	TxPortalMaxSize   int
}
