package westworld2

import (
	"fmt"
	"github.com/pkg/errors"
	"reflect"
)

type Config struct {
	poolBufferSz           int
	portalStartSz          int
	maxSegmentSz           int
	startRetxMs            int
	rttRetxOverMs          int
	treeLen                int
	readsQLen              int
	listenerRxQLen         int
	acceptQLen             int
	increaseSz             int
	dupAckThrottleFraction float64
	i                      Instrument
}

func NewDefaultConfig() *Config {
	return &Config{
		poolBufferSz:           64 * 1024,
		portalStartSz:          3 * 1024,
		maxSegmentSz:           1500,
		startRetxMs:            100,
		rttRetxOverMs:          10,
		treeLen:                1024,
		readsQLen:              1024,
		listenerRxQLen:         1024,
		acceptQLen:             1024,
		increaseSz:             1024,
		dupAckThrottleFraction: 0.95,
	}
}

func (self *Config) Load(data map[interface{}]interface{}) error {
	if v, found := data["pool_buffer_sz"]; found {
		if i, ok := v.(int); ok {
			self.poolBufferSz = i
		} else {
			return errors.New("invalid 'pool_buffer_sz' value")
		}
	}
	if v, found := data["portal_start_sz"]; found {
		if i, ok := v.(int); ok {
			self.portalStartSz = i
		} else {
			return errors.New("invalid 'portal_start_sz' value")
		}
	}
	if v, found := data["max_segment_sz"]; found {
		if i, ok := v.(int); ok {
			self.maxSegmentSz = i
		} else {
			return errors.New("invalid 'max_segment_sz' value")
		}
	}
	if v, found := data["start_retx_ms"]; found {
		if i, ok := v.(int); ok {
			self.startRetxMs = i
		} else {
			return errors.New("invalid 'start_retx_ms' value")
		}
	}
	if v, found := data["rtt_retx_over_ms"]; found {
		if i, ok := v.(int); ok {
			self.rttRetxOverMs = i
		} else {
			return errors.New("invalid 'rtt_retx_over_ms' value")
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
	if v, found := data["increase_sz"]; found {
		if i, ok := v.(int); ok {
			self.increaseSz = i
		} else {
			return errors.New("invalid 'increase_sz' value")
		}
	}
	if v, found := data["dup_ack_throttle_fraction"]; found {
		if f, ok := v.(float64); ok {
			self.dupAckThrottleFraction = f
		} else {
			return errors.New("invalid 'dup_ack_throttle_fraction' value",)
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
	out += fmt.Sprintf("\t%-20s %d\n", "pool_buffer_sz", self.poolBufferSz)
	out += fmt.Sprintf("\t%-20s %d\n", "portal_start_sz", self.portalStartSz)
	out += fmt.Sprintf("\t%-20s %d\n", "max_segment_sz", self.maxSegmentSz)
	out += fmt.Sprintf("\t%-20s %d\n", "start_retx_ms", self.startRetxMs)
	out += fmt.Sprintf("\t%-20s %d\n", "rtt_retx_over_ms", self.rttRetxOverMs)
	out += fmt.Sprintf("\t%-20s %d\n", "tree_len", self.treeLen)
	out += fmt.Sprintf("\t%-20s %d\n", "reads_q_len", self.readsQLen)
	out += fmt.Sprintf("\t%-20s %d\n", "listener_rx_q_len", self.listenerRxQLen)
	out += fmt.Sprintf("\t%-20s %d\n", "accept_q_len", self.acceptQLen)
	out += fmt.Sprintf("\t%-20s %d\n", "increase_sz", self.increaseSz)
	out += fmt.Sprintf("\t%-20s %.4f\n", "dup_ack_throttle_fraction", self.dupAckThrottleFraction)
	out += fmt.Sprintf("\t%-20s %v\n", "instrument", reflect.TypeOf(self.i))
	out += "}"
	return out
}
