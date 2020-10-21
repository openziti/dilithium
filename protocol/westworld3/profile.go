package westworld3

import (
	"github.com/michaelquigley/dilithium/cf"
	"github.com/pkg/errors"
	"reflect"
)

const profileVersion = 1

type Profile struct {
	RandomizeSeq           bool    `cf:"randomize_seq"`
	TxPortalStartSz        int     `cf:"tx_portal_start_sz"`
	TxPortalMinSz          int     `cf:"tx_portal_min_sz"`
	TxPortalMaxSz          int     `cf:"tx_portal_max_sz"`
	TxPortalIncreaseThresh int     `cf:"tx_portal_increase_thresh"`
	TxPortalIncreaseScale  float64 `cf:"tx_portal_increase_scale"`
	TxPortalDupAckThresh   int     `cf:"tx_portal_dup_ack_thresh"`
	TxPortalDupAckScale    float64 `cf:"tx_portal_dup_ack_scale"`
	TxPortalRetxThresh     int     `cf:"tx_portal_retx_thresh"`
	TxPortalRetxScale      float64 `cf:"tx_portal_retx_scale"`
	RetxStartMs            int     `cf:"retx_start_ms"`
	RetxScale              float64 `cf:"retx_scale"`
	RetxAddMs              int     `cf:"retx_add_ms"`
	RttProbeMs             int     `cf:"rtt_probe_ms"`
	RttProbeAvg            int     `cf:"rtt_probe_avg"`
	MaxSegmentSz           int     `cf:"max_segment_sz"`
	PoolBufferSz           int     `cf:"pool_buffer_sz"`
	RxBufferSz             int     `cf:"rx_buffer_sz"`
	TxBufferSz             int     `cf:"tx_buffer_sz"`
	TreeLen                int     `cf:"tree_len"`
	ReadsQueueLen          int     `cf:"reads_queue_len"`
	ListenerRxQueueLen     int     `cf:"listener_rx_queue_len"`
	AcceptQueueLen         int     `cf:"accept_queue_len"`
	i                      Instrument
}

func NewBaselineProfile() *Profile {
	return &Profile{
		RandomizeSeq:           true,
		TxPortalStartSz:        16 * 1024,
		TxPortalMinSz:          16 * 1024,
		TxPortalMaxSz:          4 * 1024 * 1024,
		TxPortalIncreaseThresh: 224,
		TxPortalIncreaseScale:  1.0,
		TxPortalDupAckThresh:   64,
		TxPortalDupAckScale:    0.9,
		TxPortalRetxThresh:     64,
		TxPortalRetxScale:      0.75,
		RetxStartMs:            200,
		RetxScale:              2.0,
		RetxAddMs:              100,
		RttProbeMs:             50,
		RttProbeAvg:            8,
		MaxSegmentSz:           1420,
		PoolBufferSz:           64 * 1024,
		RxBufferSz:             16 * 1024 * 1024,
		TxBufferSz:             16 * 1024 * 1024,
		TreeLen:                1024,
		ReadsQueueLen:          1024,
		ListenerRxQueueLen:     1024,
		AcceptQueueLen:         1024,
	}
}

func (self *Profile) Load(data map[interface{}]interface{}) error {
	if v, found := data["profile_version"]; found {
		if i, ok := v.(int); ok {
			if i != profileVersion {
				return errors.Errorf("invalid profile version [%d != %d]", i, profileVersion)
			}
		} else {
			return errors.Errorf("invalid 'profile_version' value [%s]", reflect.TypeOf(v))
		}
	} else {
		return errors.New("missing 'profile_version'")
	}
	return cf.Load(data, self)
}

func (self *Profile) Dump() string {
	return cf.Dump(reflect.TypeOf(self).String(), self)
}