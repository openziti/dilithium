package westworld2

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"net"
	"testing"
	"time"
)

func TestIpToUint32Speed(t *testing.T) {
	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:7676")
	assert.NoError(t, err)

	start := time.Now()
	for i := 0; i < 40000000; i++ {
		netIPtoUint32(addr.IP)
	}
	delta := time.Since(start).Microseconds()
	usecEach := delta / 40000000.0
	fmt.Printf("%d usec, %d usec/ip\n", delta, usecEach)
}
