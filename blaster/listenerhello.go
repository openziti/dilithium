package blaster

import (
	"bytes"
	"encoding/gob"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"time"
)

func (self *listenerConn) hello() error {
	/*
	 * Receive cconn Sync
	 */
	logrus.Infof("started receiving cconn sync")
	reqMsg := cmsg{}
	if err := self.cdec.Decode(&reqMsg); err != nil {
		_ = self.cconn.Close()
		return errors.Wrap(err, "decode cmsg")
	}
	if reqMsg.Mt != Sync {
		_ = self.cconn.Close()
		return errors.Errorf("expected sync got mt [%d]", reqMsg.Mt)
	}
	logrus.Infof("finished receiving cconn sync")
	/* */

	/*
	 * Transmit cconn Hello
	 */
	logrus.Infof("started transmitting cconn hello")
	if err := self.cenc.Encode(&cmsg{self.seq.Next(), Hello}); err != nil {
		_ = self.cconn.Close()
		return errors.Wrap(err, "encode cmsg")
	}
	if err := self.cenc.Encode(&chello{self.sessn}); err != nil {
		_ = self.cconn.Close()
		return errors.Wrap(err, "encode chello")
	}
	logrus.Infof("finished transmitting cconn hello")
	/* */

	/*
	 * Receive dconn Hello
	 */
	logrus.Infof("started receiving dconn hello")
	start := time.Now()
	success := false
	for {
		if time.Now().Sub(start).Seconds() >= 5 {
			break // hello timeout
		}

		select {
		case cp := <-self.rxq:
			if cp.h.Mt == Hello {
				if cp.p.(chello).Nonce == self.sessn {
					self.dpeer = cp.peer
					self.listn.active[self.dpeer.String()] = self
					delete(self.listn.syncing, self.sessn)
					success = true
					break
				}
			}

		case <-time.After(5 * time.Second):
			break // packet timeout
		}
	}
	if success {
		if err := self.cenc.Encode(&cmsg{self.seq.Next(), Ok}); err != nil {
			return errors.Wrap(err, "encode ok")
		}
	} else {
		return errors.New("dconn hello timeout")
	}
	logrus.Infof("finished receiving dconn hello")
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

	logrus.Infof("connection established")

	return nil
}

func (self *listenerConn) helloTxDconn(closer chan struct{}) {
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
