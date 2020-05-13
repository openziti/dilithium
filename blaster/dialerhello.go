package blaster

import (
	"bytes"
	"encoding/gob"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"time"
)

func (self *dialerConn) hello() error {
	/*
	 * Transmit cconn Sync
	 */
	logrus.Infof("starting cconn sync")
	if err := self.cenc.Encode(&cmsg{self.seq.Next(), Sync}); err != nil {
		_ = self.cconn.Close()
		_ = self.dconn.Close()
		return errors.Wrap(err, "encode sync")
	}
	logrus.Infof("finished cconn sync")
	/* */

	/*
	 * Receive cconn Hello
	 */
	logrus.Infof("starting cconn hello")
	rplMsg := cmsg{}
	if err := self.cdec.Decode(&rplMsg); err != nil {
		_ = self.cconn.Close()
		_ = self.dconn.Close()
		return errors.Wrap(err, "decode cmsg")
	}
	if rplMsg.Mt != Hello {
		_ = self.cconn.Close()
		_ = self.dconn.Close()
		return errors.Errorf("expected hello, Mt [%d]", rplMsg.Mt)
	}
	rplHello := chello{}
	if err := self.cdec.Decode(&rplHello); err != nil {
		_ = self.cconn.Close()
		_ = self.dconn.Close()
		return errors.Wrap(err, "decode chello")
	}
	self.sessn = rplHello.Nonce
	logrus.Infof("hello session [%s]", self.sessn)
	logrus.Infof("finished cconn hello")
	/* */

	/*
	 * Transmit dconn Hello
	 */
	logrus.Infof("starting transmitting dconn hello")
	closer := make(chan struct{}, 1)
	defer func() { close(closer) }()
	go self.helloTxDconn(closer)

	if err := self.cconn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return errors.Wrap(err, "set cconn deadline")
	}

	okMsg := cmsg{}
	if err := self.cdec.Decode(&okMsg); err != nil {
		return errors.Wrap(err, "decode ok")
	}
	if okMsg.Mt != Ok {
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
	cp, err := self.readCmsgPair()
	if err != nil {
		return errors.Wrap(err, "read cmsgPair")
	}
	if cp.h.Mt != Hello {
		return errors.Errorf("expected hello mt [%d]", cp.h.Mt)
	}
	if cp.p.(chello).Nonce != self.sessn {
		return errors.New("invalid session")
	}
	if err := self.cenc.Encode(&cmsg{self.seq.Next(), Ok}); err != nil {
		return errors.Wrap(err, "encode ok")
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