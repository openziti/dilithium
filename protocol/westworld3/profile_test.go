package westworld3

import (
	"fmt"
	"github.com/openziti/dilithium/cf"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestProfileLoad(t *testing.T) {
	p := NewBaselineProfile()
	d := make(map[string]interface{})
	d["randomize_seq"] = true
	d["tx_portal_start_sz"] = 17 * 1024
	d["tx_portal_dupack_capacity_scale"] = 4.5
	assert.False(t, p.RandomizeSeq)
	assert.Equal(t, 16 * 1024, p.TxPortalStartSz)
	assert.Equal(t, 0.9, p.TxPortalDupAckCapacityScale)
	err := cf.Load(d, p)
	assert.NoError(t, err)
	assert.True(t, p.RandomizeSeq)
	assert.Equal(t, 17 * 1024, p.TxPortalStartSz)
	assert.Equal(t, 4.5, p.TxPortalDupAckCapacityScale)
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