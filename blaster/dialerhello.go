package blaster

import (
	"bytes"
	"encoding/gob"
	"github.com/michaelquigley/dilithium/blaster/pb"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"time"
)

func (self *dialerConn) hello() error {
	/*
	 * Transmit cconn Sync
	 */
	logrus.Infof("started transmitting cconn sync")
	err := pb.WriteMessage(pb.NewSync(self.seq.Next()), self.cconn)
	if err != nil {
		return errors.Wrap(err, "sync write")
	}
	logrus.Infof("finished transmitting cconn sync")
	/* */

	/*
	 * Receive cconn Hello
	 */
	logrus.Infof("started receiving cconn hello")
	wm, err := pb.ReadMessage(self.cconn)
	if err != nil {
		return errors.Wrap(err, "hello read")
	}
	if wm.Type != pb.MessageType_HELLO {
		return errors.Errorf("expected hello mt [%d]", wm.Type)
	}
	self.sessn = wm.HelloPayload.Session
	logrus.Infof("hello session [%s]", self.sessn)
	logrus.Infof("finished receiving cconn hello")
	/* */

	/*
	 * Transmit dconn Hello
	 */
	logrus.Infof("started transmitting dconn hello")
	closer := make(chan struct{}, 1)
	defer func() { close(closer) }()
	go self.helloTxDconn(closer)

	if err := self.cconn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return errors.Wrap(err, "set cconn deadline")
	}
	wm, err = pb.ReadMessage(self.cconn)
	if err != nil {
		return errors.Wrap(err, "ok read")
	}
	if wm.Type != pb.MessageType_OK {
		return errors.New("not ok")
	}
	if err := self.cconn.SetReadDeadline(time.Time{}); err != nil {
		return errors.Wrap(err, "clear cconn deadline")
	}
	logrus.Infof("finished transmitting dconn hello")
	/* */

	/*
	 * Receive dconn Hello
	 */
	logrus.Infof("started receiving dconn hello")
	if err := self.dconn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return errors.Wrap(err, "set dconn deadline")
	}
	wmp, err := self.readWireMessagePeer()
	if err != nil {
		return errors.Wrap(err, "read wmp")
	}
	if wmp.WireMessage.Type != pb.MessageType_HELLO {
		return errors.Errorf("expected hello mt [%d]", wm.Type)
	}
	if wmp.WireMessage.HelloPayload.Session != self.sessn {
		return errors.New("invalid session")
	}
	err = pb.WriteMessage(pb.NewOk(self.seq.Next()), self.cconn)
	if err != nil {
		return errors.Wrap(err, "ok write")
	}
	if err := self.dconn.SetReadDeadline(time.Time{}); err != nil {
		return errors.Wrap(err, "clear dconn deadline")
	}
	logrus.Infof("finished receiving dconn hello")
	/* */

	logrus.Infof("connection established")

	return nil
}

func (self *dialerConn) helloTxDconn(closer chan struct{}) {
	logrus.Infof("started")
	defer logrus.Warnf("exited")

	buffer := new(bytes.Buffer)
	enc := gob.NewEncoder(buffer)
	if err := enc.Encode(&cmsg{self.seq.Next(), Hello}); err != nil {
		logrus.Errorf("error encoding cmsg (%v)", err)
		return
	}
	if err := enc.Encode(&chello{self.sessn}); err != nil {
		logrus.Errorf("error encoding chello (%v)", err)
		return
	}
	_, err := self.dconn.WriteToUDP(buffer.Bytes(), self.dpeer)
	if err != nil {
		logrus.Errorf("error writing (%v)", err)
		return
	}
	logrus.Infof("transmitted dconn hello attempt")

	for {
		select {
		case <-time.After(1 * time.Second):
			buffer.Reset()
			if err := enc.Encode(&cmsg{self.seq.Next(), Hello}); err != nil {
				logrus.Errorf("error encoding cmsg (%v)", err)
				return
			}
			if err := enc.Encode(&chello{self.sessn}); err != nil {
				logrus.Errorf("error encoding chello (%v)", err)
				return
			}
			_, err := self.dconn.WriteToUDP(buffer.Bytes(), self.dpeer)
			if err != nil {
				logrus.Errorf("error writing (%v)", err)
				return
			}
			logrus.Infof("transmitted dconn hello attempt")

		case <-closer:
			return
		}
	}
}