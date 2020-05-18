package blaster

import (
	"github.com/michaelquigley/dilithium/protocol/blaster/pb"
	"github.com/michaelquigley/dilithium/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"net"
	"time"
)

type dialerConn struct {
	session  string
	cConn    *net.TCPConn
	cSeq     *util.Sequence
	dConn    *net.UDPConn
	dPeer    *net.UDPAddr
	dSeq     *util.Sequence
	rxWindow *rxWindow
	txWindow *txWindow
}

func newDialerConn(cConn *net.TCPConn, dConn *net.UDPConn, dPeer *net.UDPAddr) *dialerConn {
	dc := &dialerConn{
		cConn: cConn,
		cSeq:  util.NewSequence(),
		dConn: dConn,
		dPeer: dPeer,
		dSeq:  util.NewSequence(),
	}
	dc.rxWindow = newRxWindow(dc.cConn, dc.cSeq)
	dc.txWindow = newTxWindow(dc.cConn, dc.cSeq, dc.dConn, dc.dPeer)
	return dc
}

func (self *dialerConn) Read(p []byte) (n int, err error) {
	if n, err = self.rxWindow.read(p); err == nil && n > 0 {
		logrus.Infof("<- +[%d]", n)
		return
	}

	for {
		var awm *pb.AddressedWireMessage
		awm, err = self.readWireMessage()
		if err != nil {
			return 0, errors.Wrap(err, "read wire message")
		}

		if awm.WireMessage.Type == pb.MessageType_DATA {
			logrus.Infof("<- {#%d}[%d] <-", awm.WireMessage.Sequence, len(awm.WireMessage.DataPayload.Data))

			if err = self.rxWindow.rx(awm.WireMessage); err != nil {
				logrus.Errorf("rxWindow.rx (%v)", err)
			}

			if n, err = self.rxWindow.read(p); err == nil && n > 0 {
				logrus.Infof("<- [%d]", n)
				return
			}

		} else {
			return 0, errors.Errorf("invalid message type [%s]", awm.WireMessage.Type)
		}
	}
}

func (self *dialerConn) Write(p []byte) (n int, err error) {
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

func (self *dialerConn) Close() error {
	_ = self.dConn.Close()
	return self.cConn.Close()
}

func (self *dialerConn) RemoteAddr() net.Addr {
	return self.cConn.RemoteAddr()
}

func (self *dialerConn) LocalAddr() net.Addr {
	return self.cConn.LocalAddr()
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

func (self *dialerConn) cRxer() {
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

func (self *dialerConn) readWireMessage() (*pb.AddressedWireMessage, error) {
	buffer := make([]byte, 64*1024)
	n, peer, err := self.dConn.ReadFromUDP(buffer)
	if err != nil {
		return nil, errors.Wrap(err, "read error")
	}

	wm, err := pb.FromData(buffer[:n])
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal")
	}

	return &pb.AddressedWireMessage{WireMessage: wm, FromPeer: peer}, nil
}
