package westworld3

import (
	"fmt"
	"github.com/michaelquigley/dilithium/cf"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestProfileLoad(t *testing.T) {
	p := NewBaselineProfile()
	d := make(map[interface{}]interface{})
	d["randomize_seq"] = false
	d["tx_portal_start_sz"] = 17 * 1024
	d["tx_portal_dup_ack_scale"] = 4.5
	assert.True(t, p.RandomizeSeq)
	assert.Equal(t, 16 * 1024, p.TxPortalStartSz)
	assert.Equal(t, 0.9, p.TxPortalDupAckScale)
	err := cf.Load(d, p)
	assert.NoError(t, err)
	assert.False(t, p.RandomizeSeq)
	assert.Equal(t, 17 * 1024, p.TxPortalStartSz)
	assert.Equal(t, 4.5, p.TxPortalDupAckScale)
	fmt.Println(p.Dump())
}

func TestAddProfile(t *testing.T) {
	p := NewBaselineProfile()
	id, err := AddProfile(p)
	assert.NoError(t, err)
	assert.Equal(t, byte(1), id)
	id, err = AddProfile(p)
	assert.NoError(t, err)
	assert.Equal(t, byte(2), id)
}