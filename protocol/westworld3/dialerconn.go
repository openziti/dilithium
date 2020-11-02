package westworld3

import (
	"crypto/rand"
	"github.com/michaelquigley/dilithium/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"math"
	"math/big"
	"net"
	"time"
)

type dialerConn struct {
	conn     *net.UDPConn
	peer     *net.UDPAddr
	seq      *util.Sequence
	txPortal *txPortal
	rxPortal *rxPortal
	pool     *pool
	profile  *Profile
}

func newDialerConn(conn *net.UDPConn, peer *net.UDPAddr, profile *Profile) (*dialerConn, error) {
	sSeq := int64(0)
	if profile.RandomizeSeq {
		randSeq, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt32))
		if err != nil {
			return nil, errors.Wrap(err, "random sequence")
		}
		sSeq = randSeq.Int64()
	}
	dc := &dialerConn{
		conn: conn,
		peer: peer,
		seq:  util.NewSequence(int32(sSeq)),
	}
	dc.pool = newPool("dialerConn", uint32(dataStart+profile.MaxSegmentSz), nil)
	dc.txPortal = newTxPortal(conn, peer, profile, nil)
	dc.rxPortal = newRxPortal(conn, peer, dc.txPortal, dc.seq, profile)
	go dc.rxer()
	return dc, nil
}

func (self *dialerConn) Read(p []byte) (int, error) {
	return self.rxPortal.read(p)
}

func (self *dialerConn) Write(p []byte) (int, error) {
	return self.txPortal.tx(p, self.seq)
}

func (self *dialerConn) Close() error {
	logrus.Infof("close requested")
	return self.txPortal.close(self.seq)
}

func (self *dialerConn) RemoteAddr() net.Addr {
	return self.peer
}

func (self *dialerConn) LocalAddr() net.Addr {
	return self.conn.LocalAddr()
}

func (self *dialerConn) SetDeadline(t time.Time) error {
	return nil
}

func (self *dialerConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (self *dialerConn) SetWriteDeadline(t time.Time) error {
	return nil
}

func (self *dialerConn) rxer() {
	logrus.Infof("started")
	defer logrus.Warn("exited")
	//
}

func (self *dialerConn) hello() error {
	return errors.New("not implemented")
}