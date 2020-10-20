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
