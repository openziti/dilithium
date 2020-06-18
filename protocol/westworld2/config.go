package westworld2

import (
	"fmt"
	"github.com/pkg/errors"
)

type Config struct {
	portalStartSz int
	mss           int
	retxTimeoutMs int
	treeSz        int
	readsQSz      int
	listenerRxQSz int
	acceptQSz     int
	bufferSz      int
	i             Instrument
}

func NewDefaultConfig() *Config {
	return &Config{
		portalStartSz: 3 * 1024,
		mss:           1500,
		retxTimeoutMs: 100,
		treeSz:        1024,
		readsQSz:      1024,
		listenerRxQSz: 1024,
		acceptQSz:     1024,
		bufferSz:      64 * 1024,
	}
}

func (self *Config) Load(data map[interface{}]interface{}) error {
	if v, found := data["portal_start_sz"]; found {
		if i, ok := v.(int); ok {
			self.portalStartSz = i
		} else {
			return errors.New("invalid 'portal_start_sz' value")
		}
	}
	if v, found := data["mss"]; found {
		if i, ok := v.(int); ok {
			self.mss = i
		} else {
			return errors.New("invalid 'mss' value")
		}
	}
	if v, found := data["retx_timeout_ms"]; found {
		if i, ok := v.(int); ok {
			self.retxTimeoutMs = i
		} else {
			return errors.New("invalid 'retx_timeout_ms' value")
		}
	}
	if v, found := data["tree_sz"]; found {
		if i, ok := v.(int); ok {
			self.treeSz = i
		} else {
			return errors.New("invalid 'tree_sz' value")
		}
	}
	if v, found := data["reads_q_sz"]; found {
		if i, ok := v.(int); ok {
			self.readsQSz = i
		} else {
			return errors.New("invalid 'reads_q_sz' value")
		}
	}
	if v, found := data["listener_rx_q_sz"]; found {
		if i, ok := v.(int); ok {
			self.listenerRxQSz = i
		} else {
			return errors.New("invalid 'listener_rx_q_sz' value")
		}
	}
	if v, found := data["accept_q_sz"]; found {
		if i, ok := v.(int); ok {
			self.acceptQSz = i
		} else {
			return errors.New("invalid 'accept_q_sz' value")
		}
	}
	if v, found := data["buffer_sz"]; found {
		if i, ok := v.(int); ok {
			self.bufferSz = i
		} else {
			return errors.New("invalid 'buffer_sz' value")
		}
	}
	return nil
}

func (self *Config) Dump() string {
	out := "westworld2.Config{\n"
	out += fmt.Sprintf("\t%-20s %d\n", "portalStartSz", self.portalStartSz)
	out += fmt.Sprintf("\t%-20s %d\n", "mss", self.mss)
	out += fmt.Sprintf("\t%-20s %d\n", "retxTimeoutMs", self.retxTimeoutMs)
	out += fmt.Sprintf("\t%-20s %d\n", "treeSz", self.treeSz)
	out += fmt.Sprintf("\t%-20s %d\n", "readsQSz", self.readsQSz)
	out += fmt.Sprintf("\t%-20s %d\n", "listenerRxQSz", self.listenerRxQSz)
	out += fmt.Sprintf("\t%-20s %d\n", "acceptQSz", self.acceptQSz)
	out += fmt.Sprintf("\t%-20s %d\n", "bufferSz", self.bufferSz)
	out += "}"
	return out
}