package westworld2

import (
	"fmt"
	"github.com/pkg/errors"
	"reflect"
)

type Config struct {
	txPortalStartSz        int
	txPortalIncreaseSz     int
	maxSegmentSz           int
	retxStartMs            int
	retxAddMs              int
	dupAckThrottleFraction float64
	retxThrottleFraction   float64
	poolBufferSz           int
	treeLen                int
	readsQLen              int
	listenerRxQLen         int
	acceptQLen             int
	i                      Instrument
}

func NewDefaultConfig() *Config {
	return &Config{
		txPortalStartSz:        3 * 1024,
		txPortalIncreaseSz:     128,
		maxSegmentSz:           1500,
		retxStartMs:            100,
		retxAddMs:              10,
		dupAckThrottleFraction: 0.95,
		retxThrottleFraction:   0.95,
		poolBufferSz:           64 * 1024,
		treeLen:                1024,
		readsQLen:              1024,
		listenerRxQLen:         1024,
		acceptQLen:             1024,
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
	if v, found := data["tx_portal_increase_sz"]; found {
		if i, ok := v.(int); ok {
			self.txPortalIncreaseSz = i
		} else {
			return errors.New("invalid 'tx_portal_increase_sz' value")
		}
	}
	if v, found := data["max_segment_sz"]; found {
		if i, ok := v.(int); ok {
			self.maxSegmentSz = i
		} else {
			return errors.New("invalid 'max_segment_sz' value")
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
	if v, found := data["dup_ack_throttle_fraction"]; found {
		if f, ok := v.(float64); ok {
			self.dupAckThrottleFraction = f
		} else {
			return errors.New("invalid 'dup_ack_throttle_fraction' value")
		}
	}
	if v, found := data["retx_throttle_fraction"]; found {
		if f, ok := v.(float64); ok {
			self.retxThrottleFraction = f
		} else {
			return errors.New("invalid 'retx_throttle_fraction' value")
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
	out += fmt.Sprintf("\t%-30s %d\n", "pool_buffer_sz", self.poolBufferSz)
	out += fmt.Sprintf("\t%-30s %d\n", "tx_portal_start_sz", self.txPortalStartSz)
	out += fmt.Sprintf("\t%-30s %d\n", "max_segment_sz", self.maxSegmentSz)
	out += fmt.Sprintf("\t%-30s %d\n", "retx_start_ms", self.retxStartMs)
	out += fmt.Sprintf("\t%-30s %d\n", "retx_add_ms", self.retxAddMs)
	out += fmt.Sprintf("\t%-30s %d\n", "tree_len", self.treeLen)
	out += fmt.Sprintf("\t%-30s %d\n", "reads_q_len", self.readsQLen)
	out += fmt.Sprintf("\t%-30s %d\n", "listener_rx_q_len", self.listenerRxQLen)
	out += fmt.Sprintf("\t%-30s %d\n", "accept_q_len", self.acceptQLen)
	out += fmt.Sprintf("\t%-30s %d\n", "increase_sz", self.txPortalIncreaseSz)
	out += fmt.Sprintf("\t%-30s %.4f\n", "dup_ack_throttle_fraction", self.dupAckThrottleFraction)
	out += fmt.Sprintf("\t%-30s %.4f\n", "retx_throttle_fraction", self.retxThrottleFraction)
	out += fmt.Sprintf("\t%-30s %v\n", "instrument", reflect.TypeOf(self.i))
	out += "}"
	return out
}
