package blaster

import (
	"github.com/michaelquigley/dilithium/protocol/blaster/pb"
	"github.com/michaelquigley/dilithium/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"net"
	"time"
)

type listenerConn struct {
	cListener *listener
	session   string
	cConn     net.Conn
	cSeq      *util.Sequence
	dConn     *net.UDPConn
	dPeer     *net.UDPAddr
	dSeq      *util.Sequence
	dRxQueue  chan *pb.AddressedWireMessage
	rxWindow  *rxWindow
	txWindow  *txWindow
}

func newListenerConn(cListener *listener, session string, cConn net.Conn, dConn *net.UDPConn) *listenerConn {
	lc := &listenerConn{
		cListener: cListener,
		session:   session,
		cConn:     cConn,
		cSeq:      util.NewSequence(0),
		dConn:     dConn,
		dSeq:      util.NewSequence(0),
		dRxQueue:  make(chan *pb.AddressedWireMessage, 1024),
	}
	lc.rxWindow = newRxWindow(lc.cConn, lc.cSeq)
	lc.txWindow = newTxWindow(lc.cConn, lc.cSeq, lc.dConn, lc.dPeer)
	return lc
}

func (self *listenerConn) Read(p []byte) (n int, err error) {
	if n, err = self.rxWindow.read(p); err == nil && n > 0 {
		logrus.Infof("<- +[%d]", n)
		return
	}

	for {
		if awm, ok := <-self.dRxQueue; ok {
			if awm.WireMessage.Type == pb.MessageType_DATA {
				logrus.Infof(" <- {#%d}[%d]", awm.WireMessage.Sequence, len(awm.WireMessage.DataPayload.Data))

				if err := self.rxWindow.rx(awm.WireMessage); err != nil {
					logrus.Errorf("rxWindow.rx (%v)", err)
				}

				if n, err = self.rxWindow.read(p); err == nil && n > 0 {
					logrus.Infof("<- [%d]", n)
					return
				}

			} else {
				return 0, errors.Errorf("unexpected message [%s]", awm.WireMessage.Type.String())
			}
		} else {
			return 0, errors.New("closed")
		}

	}
}

func (self *listenerConn) Write(p []byte) (n int, err error) {
	logrus.Infof("[%d] ->", len(p))

	wm := pb.NewData(self.dSeq.Next(), p)
	self.txWindow.tx(wm)

	data, err := pb.ToData(wm)
	if err != nil {
		return 0, errors.Wrap(err, "encode data")
	}
	_, err = self.dConn.WriteToUDP(data, self.dPeer)
	if err != nil {
		return 0, errors.Wrap(err, "write data")
	}
	logrus.Infof("{#%d}[%d] ->", wm.Sequence, len(wm.DataPayload.Data))
	return len(wm.DataPayload.Data), nil
}

func (self *listenerConn) Close() error {
	return self.cConn.Close()
}

func (self *listenerConn) RemoteAddr() net.Addr {
	return self.cConn.RemoteAddr()
}

func (self *listenerConn) LocalAddr() net.Addr {
	return self.cConn.LocalAddr()
}

func (self *listenerConn) SetDeadline(t time.Time) error {
	return nil
}

func (self *listenerConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (self *listenerConn) SetWriteDeadline(t time.Time) error {
	return nil
}

func (self *listenerConn) cRxer() {
	logrus.Infof("started")
	defer logrus.Warnf("exited")

	for {
		if wm, err := pb.ReadMessage(self.cConn); err == nil {
			if wm.Type == pb.MessageType_ACK {
				self.txWindow.ack(wm.AckPayload.HighWater, wm.AckPayload.Missing)

			} else if wm.Type == pb.MessageType_EOW {
				self.rxWindow.eow(wm.EowPayload.HighWater)

			} else {
				logrus.Warnf("no handler for cConn msg [%s]", wm.Type.String())
			}
		}
	}
}

func (self *listenerConn) dQueue(awm *pb.AddressedWireMessage) {
	self.dRxQueue <- awm
}
