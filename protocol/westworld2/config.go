package westworld2

import (
	"fmt"
	"github.com/pkg/errors"
	"reflect"
)

type Config struct {
	txPortalStartSz      int
	txPortalMinSz        int
	txPortalMaxSz        int
	txPortalIncreaseCt   int
	txPortalIncreaseFrac float64
	txPortalDupAckCt     int
	txPortalDupAckFrac   float64
	txPortalRetxCt       int
	txPortalRetxFrac     float64
	retxStartMs          int
	retxAddMs            int
	rttProbeMs			 int
	rttProbeAvg          int
	maxSegmentSz         int
	poolBufferSz         int
	treeLen              int
	readsQLen            int
	listenerRxQLen       int
	acceptQLen           int
	i                    Instrument
}

func NewDefaultConfig() *Config {
	return &Config{
		txPortalStartSz:      3 * 1024,
		txPortalMinSz:        2048,
		txPortalMaxSz:        1024 * 1024,
		txPortalIncreaseCt:   1,
		txPortalIncreaseFrac: 0.1,
		txPortalDupAckCt:     1,
		txPortalDupAckFrac:   0.95,
		txPortalRetxCt:       1,
		txPortalRetxFrac:     0.95,
		retxStartMs:          100,
		retxAddMs:            10,
		rttProbeMs:			  1000,
		rttProbeAvg:		  8,
		maxSegmentSz:         1500,
		poolBufferSz:         64 * 1024,
		treeLen:              1024,
		readsQLen:            1024,
		listenerRxQLen:       1024,
		acceptQLen:           1024,
	}
}

func (self *Config) Load(data map[interface{}]interface{}) error {
	if v, found := data["tx_portal_start_sz"]; found {
		if i, ok := v.(int); ok {
			self.txPortalStartSz = i
		} else {
			return errors.New("invalid 'tx_portal_start_sz' value")
		}
	}
	if v, found := data["tx_portal_min_sz"]; found {
		if i, ok := v.(int); ok {
			self.txPortalMinSz = i
		} else {
			return errors.New("invalid 'tx_portal_min_sz' value")
		}
	}
	if v, found := data["tx_portal_max_sz"]; found {
		if i, ok := v.(int); ok {
			self.txPortalMaxSz = i
		} else {
			return errors.New("invalid 'tx_portal_max_sz' value")
		}
	}
	if v, found := data["tx_portal_increase_ct"]; found {
		if i, ok := v.(int); ok {
			self.txPortalIncreaseCt = i
		} else {
			return errors.New("invalid 'tx_portal_increase_ct' value")
		}
	}
	if v, found := data["tx_portal_increase_frac"]; found {
		if f, ok := v.(float64); ok {
			self.txPortalIncreaseFrac = f
		} else {
			return errors.New("invalid 'tx_portal_increase_frac' value")
		}
	}
	if v, found := data["tx_portal_dup_ack_ct"]; found {
		if i, ok := v.(int); ok {
			self.txPortalDupAckCt = i
		} else {
			return errors.New("invalid 'tx_portal_dup_ack_ct' value")
		}
	}
	if v, found := data["tx_portal_dup_ack_frac"]; found {
		if f, ok := v.(float64); ok {
			self.txPortalDupAckFrac = f
		} else {
			return errors.New("invalid 'tx_portal_dup_ack_frac' value")
		}
	}
	if v, found := data["tx_portal_retx_ct"]; found {
		if i, ok := v.(int); ok {
			self.txPortalRetxCt = i
		} else {
			return errors.New("invalid 'tx_portal_retx_ct' value")
		}
	}
	if v, found := data["tx_portal_retx_frac"]; found {
		if f, ok := v.(float64); ok {
			self.txPortalRetxFrac = f
		} else {
			return errors.New("invalid 'tx_portal_retx_frac' value")
		}
	}
	if v, found := data["retx_start_ms"]; found {
		if i, ok := v.(int); ok {
			self.retxStartMs = i
		} else {
			return errors.New("invalid 'retx_start_ms' value")
		}
	}
	if v, found := data["retx_add_ms"]; found {
		if i, ok := v.(int); ok {
			self.retxAddMs = i
		} else {
			return errors.New("invalid 'retx_add_ms' value")
		}
	}
	if v, found := data["rtt_probe_ms"]; found {
		if i, ok := v.(int); ok {
			self.rttProbeMs = i
		} else {
			return errors.New("invalid 'rtt_probe_ms' value")
		}
	}
	if v, found := data["rtt_probe_avg"]; found {
		if i, ok := v.(int); ok {
			self.rttProbeAvg = i
		} else {
			return errors.New("invalid 'rtt_probe_avg' value")
		}
	}
	if v, found := data["max_segment_sz"]; found {
		if i, ok := v.(int); ok {
			self.maxSegmentSz = i
		} else {
			return errors.New("invalid 'max_segment_sz' value")
		}
	}
	if v, found := data["pool_buffer_sz"]; found {
		if i, ok := v.(int); ok {
			self.poolBufferSz = i
		} else {
			return errors.New("invalid 'pool_buffer_sz' value")
		}
	}
	if v, found := data["tree_len"]; found {
		if i, ok := v.(int); ok {
			self.treeLen = i
		} else {
			return errors.New("invalid 'tree_len' value")
		}
	}
	if v, found := data["reads_q_len"]; found {
		if i, ok := v.(int); ok {
			self.readsQLen = i
		} else {
			return errors.New("invalid 'reads_q_len' value")
		}
	}
	if v, found := data["listener_rx_q_len"]; found {
		if i, ok := v.(int); ok {
			self.listenerRxQLen = i
		} else {
			return errors.New("invalid 'listener_rx_q_len' value")
		}
	}
	if v, found := data["accept_q_len"]; found {
		if i, ok := v.(int); ok {
			self.acceptQLen = i
		} else {
			return errors.New("invalid 'accept_q_len' value")
		}
	}
	if v, found := data["instrument"]; found {
		if submap, ok := v.(map[string]interface{}); ok {
			if v, found := submap["name"]; found {
				if name, ok := v.(string); ok {
					i, err := NewInstrument(name, submap)
					if err != nil {
						return errors.Wrap(err, "error creating instrument")
					}
					self.i = i
				} else {
					return errors.New("invalid 'instrument/name' value")
				}
			} else {
				return errors.New("missing 'instrument/name'")
			}
		} else {
			return errors.Errorf("invalid 'instrument' value [%v]", reflect.TypeOf(v))
		}
	}
	return nil
}

func (self *Config) Dump() string {
	out := "westworld2.Config{\n"
	out += fmt.Sprintf("\t%-30s %d\n", "tx_portal_start_sz", self.txPortalStartSz)
	out += fmt.Sprintf("\t%-30s %d\n", "tx_portal_min_sz", self.txPortalMinSz)
	out += fmt.Sprintf("\t%-30s %d\n", "tx_portal_max_sz", self.txPortalMaxSz)
	out += fmt.Sprintf("\t%-30s %d\n", "tx_portal_incrase_ct", self.txPortalIncreaseCt)
	out += fmt.Sprintf("\t%-30s %.4f\n", "tx_portal_increase_frac", self.txPortalIncreaseFrac)
	out += fmt.Sprintf("\t%-30s %d\n", "tx_portal_dup_ack_ct", self.txPortalDupAckCt)
	out += fmt.Sprintf("\t%-30s %.4f\n", "tx_portal_dup_ack_frac", self.txPortalDupAckFrac)
	out += fmt.Sprintf("\t%-30s %d\n", "tx_portal_retx_ct", self.txPortalRetxCt)
	out += fmt.Sprintf("\t%-30s %.4f\n", "tx_portal_retx_frac", self.txPortalRetxFrac)
	out += fmt.Sprintf("\t%-30s %d\n", "retx_start_ms", self.retxStartMs)
	out += fmt.Sprintf("\t%-30s %d\n", "retx_add_ms", self.retxAddMs)
	out += fmt.Sprintf("\t%-30s %d\n", "rtt_probe_ms", self.rttProbeMs)
	out += fmt.Sprintf("\t%-30s %d\n", "rtt_probe_avg", self.rttProbeAvg)
	out += fmt.Sprintf("\t%-30s %d\n", "max_segment_sz", self.maxSegmentSz)
	out += fmt.Sprintf("\t%-30s %d\n", "pool_buffer_sz", self.poolBufferSz)
	out += fmt.Sprintf("\t%-30s %d\n", "tree_len", self.treeLen)
	out += fmt.Sprintf("\t%-30s %d\n", "reads_q_len", self.readsQLen)
	out += fmt.Sprintf("\t%-30s %d\n", "listener_rx_q_len", self.listenerRxQLen)
	out += fmt.Sprintf("\t%-30s %d\n", "accept_q_len", self.acceptQLen)
	out += fmt.Sprintf("\t%-30s %v\n", "instrument", reflect.TypeOf(self.i))
	out += "}"
	return out
}
