package westworld3

import (
	"crypto/rand"
	"github.com/openziti/dilithium/util"
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

func (self *dialerConn) SetDeadline(_ time.Time) error {
	return errors.New("not implemented")
}

func (self *dialerConn) SetReadDeadline(_ time.Time) error {
	return errors.New("not implemented")
}

func (self *dialerConn) SetWriteDeadline(_ time.Time) error {
	return errors.New("not implemented")
}

func (self *dialerConn) rxer() {
	logrus.Infof("started")
	defer logrus.Warn("exited")

	for {
		wm, _, err := readWireMessage(self.conn, self.pool)
		if err != nil {
			logrus.Errorf("error reading (%v)", err)
			continue
		}

		switch wm.mt {
		case DATA:
			_, rttTs, err := wm.asData()
			if err != nil {
				logrus.Errorf("as data error (%v)", err)
				continue
			}
			if rttTs != nil {
				self.txPortal.rtt(*rttTs)
			}
			if err := self.rxPortal.rx(wm); err != nil {
				logrus.Errorf("error rx-ing (%v)", err)
				continue
			}

		case ACK:
			acks, rxPortalSz, rttTs, err := wm.asAck()
			if err != nil {
				logrus.Errorf("as ack error (%v)", err)
				continue
			}
			if rttTs != nil {
				self.txPortal.rtt(*rttTs)
			}
			self.txPortal.updateRxPortalSz(int(rxPortalSz))
			if err := self.txPortal.ack(acks); err != nil {
				logrus.Errorf("error acking (%v)", err)
				continue
			}

		case CLOSE:
			if err := self.rxPortal.rx(wm); err != nil {
				logrus.Errorf("error rx-ing close (%v)", err)
			}
		}
	}
}

func (self *dialerConn) hello() error {
	helloSeq := self.seq.Next()
	hello, err := newHello(helloSeq, hello{protocolVersion, 0}, nil, self.pool)
	if err != nil {
		return errors.Wrap(err, "error creating hello message")
	}
	defer hello.buffer.unref()

	count := 0
	for {
		if err := writeWireMessage(hello, self.conn, self.peer); err != nil {
			return errors.Wrap(err, "write hello")
		}

		if err := self.conn.SetReadDeadline(time.Now().Add(time.Duration(self.profile.ConnectionTimeoutMs) * time.Millisecond)); err != nil {
			return errors.Wrap(err, "set read deadline")
		}

		helloAck, _, err := readWireMessage(self.conn, self.pool)
		if err != nil {
			return errors.Wrap(err, "read hello ack")
		}
		defer helloAck.buffer.unref()

		if err := self.conn.SetReadDeadline(time.Time{}); err != nil {
			return errors.Wrap(err, "clear read deadline")
		}

		h, acks, err := helloAck.asHello()
		if err != nil {
			return errors.Wrap(err, "unexpected response")
		}

		if h.version != protocolVersion {
			return errors.New("unexpected protocol version")
		}

		if len(acks) == 1 && acks[0].start == acks[0].end && acks[0].start == helloSeq {
			// Set next highest sequence
			self.rxPortal.setAccepted(helloAck.seq)

			finalAcks := []ack{{helloAck.seq, helloAck.seq}}
			finalAck, err := newAck(finalAcks, 0, nil, self.pool)
			if err != nil {
				return errors.Wrap(err, "new final ack")
			}
			if err := writeWireMessage(finalAck, self.conn, self.peer); err != nil {
				return errors.Wrap(err, "write final ack")
			}

			go self.rxer()
			return nil
		}

		count++
		if count > 5 {
			return errors.New("connection timeout")
		}
	}
}