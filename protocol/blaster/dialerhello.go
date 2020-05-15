package blaster

import (
	"github.com/michaelquigley/dilithium/protocol/blaster/pb"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"time"
)

func (self *dialerConn) hello() error {
	/*
	 * Tx cConn Sync
	 */
	logrus.Infof("start tx cConn sync")
	err := pb.WriteMessage(pb.NewSync(self.cSeq.Next()), self.cConn)
	if err != nil {
		return errors.Wrap(err, "sync write")
	}
	/* */

	/*
	 * Rx cConn Hello
	 */
	logrus.Infof("start rx cConn hello")
	wm, err := pb.ReadMessage(self.cConn)
	if err != nil {
		return errors.Wrap(err, "hello read")
	}
	if wm.Type != pb.MessageType_HELLO {
		return errors.Errorf("expected hello mt [%d]", wm.Type)
	}
	self.session = wm.HelloPayload.Session
	logrus.Infof("hello [%s]", self.session)
	/* */

	/*
	 * Tx dConn Hello
	 */
	logrus.Infof("start tx dConn hello")
	closer := make(chan struct{}, 1)
	defer func() { close(closer) }()
	go self.helloTxDConn(closer)

	if err := self.cConn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return errors.Wrap(err, "set cConn deadline")
	}
	wm, err = pb.ReadMessage(self.cConn)
	if err != nil {
		return errors.Wrap(err, "ok read")
	}
	if wm.Type != pb.MessageType_OK {
		return errors.New("not ok")
	}
	if err := self.cConn.SetReadDeadline(time.Time{}); err != nil {
		return errors.Wrap(err, "clear cConn deadline")
	}
	/* */

	/*
	 * Rx dConn Hello
	 */
	logrus.Infof("start rx dConn hello")
	if err := self.dConn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return errors.Wrap(err, "set dConn hello deadline")
	}
	awp, err := self.readWireMessage()
	if err != nil {
		return errors.Wrap(err, "read awp")
	}
	if awp.WireMessage.Type != pb.MessageType_HELLO {
		return errors.Errorf("expected hello not [%s]", wm.Type.String())
	}
	if awp.WireMessage.HelloPayload.Session != self.session {
		return errors.New("invalid session")
	}
	err = pb.WriteMessage(pb.NewOk(self.cSeq.Next()), self.cConn)
	if err != nil {
		return errors.Wrap(err, "ok write")
	}
	if err := self.dConn.SetReadDeadline(time.Time{}); err != nil {
		return errors.Wrap(err, "clear dConn deadline")
	}
	/* */

	logrus.Infof("connection established")

	return nil
}

func (self *dialerConn) helloTxDConn(closer chan struct{}) {
	logrus.Infof("started")
	defer logrus.Warnf("exited")

	data, err := pb.ToData(pb.NewHello(-1, self.session))
	if err != nil {
		logrus.Errorf("error encoding hello (%v)", err)
		return
	}
	n, err := self.dConn.WriteToUDP(data, self.dPeer)
	if err != nil {
		logrus.Errorf("error writing (%v)", err)
		return
	}
	if n != len(data) {
		logrus.Errorf("short write")
		return
	}
	logrus.Infof("transmitted dConn hello attempt [%s]", self.dPeer)

	for {
		select {
		case <-time.After(1 * time.Second):
			data, err := pb.ToData(pb.NewHello(-1, self.session))
			if err != nil {
				logrus.Errorf("error encoding hello (%v)", err)
				return
			}
			n, err := self.dConn.WriteToUDP(data, self.dPeer)
			if err != nil {
				logrus.Errorf("error writing (%v)", err)
				return
			}
			if n != len(data) {
				logrus.Errorf("short write")
				return
			}
			logrus.Infof("transmitted dConn hello attempt [%s]", self.dPeer)

		case <-closer:
			return
		}
	}
}