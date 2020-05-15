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
	txWindow  *txWindow
}

func newListenerConn(cListener *listener, session string, cConn net.Conn, dConn *net.UDPConn) *listenerConn {
	lc := &listenerConn{
		cListener: cListener,
		session:   session,
		cConn:     cConn,
		cSeq:      util.NewSequence(),
		dConn:     dConn,
		dSeq:      util.NewSequence(),
		dRxQueue:  make(chan *pb.AddressedWireMessage, 1024),
	}
	lc.txWindow = newTxWindow(lc.cConn, lc.cSeq, lc.dConn, lc.dPeer)
	go lc.cRxer()
	return lc
}

func (self *listenerConn) Read(p []byte) (n int, err error) {
	if awm, ok := <-self.dRxQueue; ok {
		if awm.WireMessage.Type == pb.MessageType_DATA {
			logrus.Infof("[#%d](%d) <-", awm.WireMessage.Sequence, len(awm.WireMessage.DataPayload.Data))
			n = copy(p, awm.WireMessage.DataPayload.Data)
			return n, nil
		} else {
			return 0, errors.Errorf("unexpected message [%s]", awm.WireMessage.Type.String())
		}
	} else {
		return 0, errors.New("closed")
	}
}

func (self *listenerConn) Write(p []byte) (n int, err error) {
	data, err := pb.ToData(pb.NewData(self.dSeq.Next(), p))
	if err != nil {
		return 0, errors.Wrap(err, "encode data")
	}
	_, err = self.dConn.WriteToUDP(data, self.dPeer)
	if err != nil {
		return 0, errors.Wrap(err, "write data")
	}
	return len(p), nil
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
				self.txWindow.ack(wm)
			} else if wm.Type == pb.MessageType_EOW {
				// self.rxWindow.eow(wm)
			}
		}
	}
}

func (self *listenerConn) dQueue(awm *pb.AddressedWireMessage) {
	self.dRxQueue <- awm
}
