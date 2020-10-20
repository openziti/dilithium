package westworld3

type Profile struct {
	seqRandom              bool
	txPortalStartSz        int
	txPortalMinSz          int
	txPortalMaxSz          int
	txPortalIncreaseThresh int
	txPortalIncreaseScale  float64
	txPortalDupAckThresh   int
	txPortalDupAckScale    float64
	txPortalRetxThresh     int
	txPortalRetxScale      float64
	retxStartMs            int
	retxScale              float64
	retxAddMs              int
	rttProbeMs             int
	rttProbeAvg            int
	maxSegmentSz           int
	poolBufferSz           int
	rxBufferSz             int
	txBufferSz             int
	treeLen                int
	readsQueueLen          int
	listenerRxQueueLen     int
	acceptQueueLen         int
	i                      Instrument
}

func NewBaselineProfile() *Profile {
	return &Profile{
		seqRandom:              true,
		txPortalStartSz:        16 * 1024,
		txPortalMinSz:          16 * 1024,
		txPortalMaxSz:          4 * 1024 * 1024,
		txPortalIncreaseThresh: 224,
		txPortalIncreaseScale:  1.0,
		txPortalDupAckThresh:   64,
		txPortalDupAckScale:    0.9,
		txPortalRetxThresh:     64,
		txPortalRetxScale:      0.75,
		retxStartMs:            200,
		retxScale:              2.0,
		retxAddMs:              100,
		rttProbeMs:             50,
		rttProbeAvg:            8,
		maxSegmentSz:           1420,
		poolBufferSz:           64 * 1024,
		rxBufferSz:             16 * 1024 * 1024,
		txBufferSz:             16 * 1024 * 1024,
		treeLen:                1024,
		readsQueueLen:          1024,
		listenerRxQueueLen:     1024,
		acceptQueueLen:         1024,
	}
}
