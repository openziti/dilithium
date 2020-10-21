package westworld3

import (
	"fmt"
	"github.com/pkg/errors"
)

const profileVersion = 1

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

func (self *Profile) Load(data map[interface{}]interface{}) error {
	if v, found := data["profile_version"]; found {
		if i, ok := v.(int); ok {
			if i != profileVersion {
				return errors.Errorf("invalid profile version [%d != %d]", i, profileVersion)
			}
		} else {
			return errors.New("invlaid 'profile_version' value")
		}
	} else {
		return errors.New("missing 'profile_version'")
	}
	return nil
}

func (self *Profile) Dump() string {
	out := "westworld3.Profile {\n"
	out += self.dumpValue("seq_random", self.seqRandom)
	out += self.dumpValue("tx_portal_start_sz", self.txPortalStartSz)
	out += self.dumpValue("tx_portal_min_sz", self.txPortalMinSz)
	out += self.dumpValue("tx_portal_max_sz", self.txPortalMaxSz)
	out += self.dumpValue("tx_portal_increase_thresh", self.txPortalIncreaseThresh)
	out += self.dumpValue("tx_portal_increase_scale", self.txPortalIncreaseScale)
	out += self.dumpValue("tx_portal_dup_ack_thresh", self.txPortalDupAckThresh)
	out += self.dumpValue("tx_portal_retx_thresh", self.txPortalRetxThresh)
	out += self.dumpValue("tx_portal_retx_scale", self.txPortalRetxScale)
	out += self.dumpValue("retx_start_ms", self.retxStartMs)
	out += self.dumpValue("retx_scale", self.retxScale)
	out += self.dumpValue("retx_add_ms", self.retxAddMs)
	out += self.dumpValue("rtt_probe_ms", self.rttProbeMs)
	out += self.dumpValue("rtt_probe_avg", self.rttProbeAvg)
	out += self.dumpValue("max_segment_sz", self.maxSegmentSz)
	out += self.dumpValue("pool_buffer_sz", self.poolBufferSz)
	out += self.dumpValue("rx_buffer_sz", self.rxBufferSz)
	out += self.dumpValue("tx_buffer_sz", self.txBufferSz)
	out += self.dumpValue("treeLen", self.treeLen)
	out += self.dumpValue("readsQueueLen", self.readsQueueLen)
	out += self.dumpValue("listenerRxQueueLen", self.listenerRxQueueLen)
	out += self.dumpValue("acceptQueueLen", self.acceptQueueLen)
	out += "}\n"
	return out
}

func (self *Profile) dumpValue(name string, v interface{}) string {
	var fstr string
	switch v.(type) {
	case bool:
		fstr = "%t"
	case int:
		fstr = "%d"
	case float64:
		fstr = "%.4f"
	default:
		fstr = "%v"
	}
	return fmt.Sprintf("\t%-30s "+fstr+"\n", name, v)
}
