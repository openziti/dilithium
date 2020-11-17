package westworld3

import (
	"github.com/openziti/dilithium/cf"
	"github.com/pkg/errors"
	"reflect"
)

const profileVersion = 1

var profileRegistry map[byte]*Profile

func init() {
	profileRegistry = make(map[byte]*Profile)
	profileRegistry[0] = NewBaselineProfile()
}

func AddProfile(p *Profile) (byte, error) {
	nextProfile := len(profileRegistry)
	if nextProfile > 255 {
		return 0, errors.New("profile registry full")
	}
	profileRegistry[byte(nextProfile)] = p
	return byte(nextProfile), nil
}

type Profile struct {
	RandomizeSeq                bool    `cf:"randomize_seq"`
	ConnectionTimeoutMs         int     `cf:"connection_timeout_ms"`
	TxPortalStartSz             int     `cf:"tx_portal_start_sz"`
	TxPortalMinSz               int     `cf:"tx_portal_min_sz"`
	TxPortalMaxSz               int     `cf:"tx_portal_max_sz"`
	TxPortalIncreaseThresh      int     `cf:"tx_portal_increase_thresh"`
	TxPortalIncreaseScale       float64 `cf:"tx_portal_increase_scale"`
	TxPortalDupAckThresh        int     `cf:"tx_portal_dupack_thresh"`
	TxPortalDupAckCapacityScale float64 `cf:"tx_portal_dupack_capacity_scale"`
	TxPortalDupAckSuccessScale  float64 `cf:"tx_portal_dupack_success_scale"`
	TxPortalRetxThresh          int     `cf:"tx_portal_retx_thresh"`
	TxPortalRetxCapacityScale   float64 `cf:"tx_portal_retx_capacity_scale"`
	TxPortalRetxSuccessScale    float64 `cf:"tx_portal_retx_success_scale"`
	RetxStartMs                 int     `cf:"retx_start_ms"`
	RetxScale                   float64 `cf:"retx_scale"`
	RetxAddMs                   int     `cf:"retx_add_ms"`
	RetxBatchMs                 int     `cf:"retx_batch_ms"`
	RttProbeMs                  int     `cf:"rtt_probe_ms"`
	RttProbeAvg                 int     `cf:"rtt_probe_avg"`
	RxPortalSzPacingThresh      float64 `cf:"rx_portal_sz_pacing_thresh"`
	MaxSegmentSz                int     `cf:"max_segment_sz"`
	PoolBufferSz                int     `cf:"pool_buffer_sz"`
	RxBufferSz                  int     `cf:"rx_buffer_sz"`
	TxBufferSz                  int     `cf:"tx_buffer_sz"`
	TxPortalTreeLen             int     `cf:"tx_portal_tree_len"`
	RetxMonitorTreeLen          int     `cf:"retx_monitor_tree_len"`
	RxPortalTreeLen             int     `cf:"rx_portal_tree_len"`
	ListenerPeersTreeLen        int     `cf:"listener_peers_tree_len"`
	ReadsQueueLen               int     `cf:"reads_queue_len"`
	ListenerRxQueueLen          int     `cf:"listener_rx_queue_len"`
	AcceptQueueLen              int     `cf:"accept_queue_len"`
	i                           Instrument
}

func NewBaselineProfile() *Profile {
	return &Profile{
		RandomizeSeq:                false,
		ConnectionTimeoutMs:         5000,
		TxPortalStartSz:             16 * 1024,
		TxPortalMinSz:               16 * 1024,
		TxPortalMaxSz:               4 * 1024 * 1024,
		TxPortalIncreaseThresh:      224,
		TxPortalIncreaseScale:       1.0,
		TxPortalDupAckThresh:        64,
		TxPortalDupAckCapacityScale: 0.9,
		TxPortalDupAckSuccessScale:  0.0,
		TxPortalRetxThresh:          64,
		TxPortalRetxCapacityScale:   0.75,
		TxPortalRetxSuccessScale:    0.0,
		RetxStartMs:                 200,
		RetxScale:                   2.0,
		RetxAddMs:                   100,
		RetxBatchMs:                 2,
		RttProbeMs:                  50,
		RttProbeAvg:                 8,
		RxPortalSzPacingThresh:		 0.5,
		MaxSegmentSz:                1420,
		PoolBufferSz:                64 * 1024,
		RxBufferSz:                  16 * 1024 * 1024,
		TxBufferSz:                  16 * 1024 * 1024,
		TxPortalTreeLen:             16 * 1024,
		RetxMonitorTreeLen:          64 * 1024,
		RxPortalTreeLen:             16 * 1024,
		ListenerPeersTreeLen:        1024,
		ReadsQueueLen:               1024,
		ListenerRxQueueLen:          1024,
		AcceptQueueLen:              1024,
		i:                           NewNilInstrument(),
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
	if v, found := data["instrument"]; found {
		submap, oks := v.(map[string]interface{})
		if !oks {
			if subi, oki := v.(map[interface{}]interface{}); oki {
				submap = make(map[string]interface{})
				oks = true
				for k, v := range subi {
					if s, ok := k.(string); ok {
						submap[s] = v
					} else {
						oks = false
					}
				}
			}
		}
		if oks {
			if v, found := submap["name"]; found {
				if name, ok := v.(string); ok {
					if i, err := NewInstrument(name, submap); err == nil {
						self.i = i
					} else {
						return errors.Wrap(err, "error configuring instrument")
					}
				} else {
					return errors.New("invalid 'name' field")
				}
			} else {
				return errors.New("instrument missing 'name' field")
			}
		} else {
			return errors.New("invalid instrument map")
		}
	}
	return cf.Load(data, self)
}

func (self *Profile) Dump() string {
	return cf.Dump(reflect.TypeOf(self).String(), self)
}
