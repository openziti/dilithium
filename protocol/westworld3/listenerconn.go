package westworld3

import (
	"crypto/rand"
	"fmt"
	"github.com/michaelquigley/dilithium/util"
	"github.com/pkg/errors"
	"math"
	"math/big"
	"net"
)

type listenerConn struct {
	listener *listener
	conn     *net.UDPConn
	peer     *net.UDPAddr
	rxQueue  chan *wireMessage
	seq      *util.Sequence
	// txPortal
	// rxPortal
	pool     *pool
	profile  *Profile
	ii       InstrumentInstance
}

func newListenerConn(listener *listener, conn *net.UDPConn, peer *net.UDPAddr, profile *Profile) (*listenerConn, error) {
	startSeq := int64(0)
	if profile.RandomizeSeq {
		randomSeq, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt32))
		if err != nil {
			return nil, errors.Wrap(err, "randomize sequence")
		}
		startSeq = randomSeq.Int64()
	}
	lc := &listenerConn{
		listener: listener,
		conn:     conn,
		peer:     peer,
		rxQueue:  make(chan *wireMessage, profile.ListenerRxQueueLen),
		seq:      util.NewSequence(int32(startSeq)),
		profile:  profile,
	}
	id := fmt.Sprintf("listenerConn_%s", conn.LocalAddr())
	if profile.i != nil {
		lc.ii = profile.i.newInstance(id, peer)
	}
	lc.pool = newPool(id, uint32(dataStart+profile.MaxSegmentSz), lc.ii)
	// txPortal
	// rxPortal
	return lc, nil
}

func (self *listenerConn) queue(wm *wireMessage) {
	self.rxQueue <- wm
}