package blaster

import (
	"github.com/michaelquigley/dilithium/blaster/pb"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"time"
)

func (self *listenerConn) hello() error {
	/*
	 * Receive cconn Sync
	 */
	logrus.Infof("started receiving cconn sync")
	wireMessage, err := pb.ReadMessage(self.cconn)
	if err != nil {
		return errors.Wrap(err, "read sync")
	}
	if wireMessage.Type != pb.MessageType_SYNC {
		return errors.Errorf("expected sync got mt [%d]", wireMessage.Type)
	}
	logrus.Infof("finished receiving cconn sync")
	/* */

	/*
	 * Transmit cconn Hello
	 */
	logrus.Infof("started transmitting cconn hello")
	err = pb.WriteMessage(pb.NewHello(self.seq.Next(), self.sessn), self.cconn)
	if err != nil {
		return errors.Wrap(err, "write hello")
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
		case wmp := <-self.rxq:
			if wmp.WireMessage.Type == pb.MessageType_HELLO {
				if wmp.WireMessage.HelloPayload.Session == self.sessn {
					self.dpeer = wmp.Peer
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
		err = pb.WriteMessage(pb.NewOk(self.seq.Next()), self.cconn)
		if err != nil {
			return errors.Wrap(err, "ok write")
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

	wmp, err := pb.ReadMessage(self.cconn)
	if err != nil {
		return errors.Wrap(err, "ok decode")
	}
	if wmp.Type != pb.MessageType_OK {
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

	data, err := pb.ToData(pb.NewHello(self.seq.Next(), self.sessn))
	if err != nil {
		logrus.Errorf("error encoding hello (%v)", err)
		return
	}
	n, err := self.dconn.WriteToUDP(data, self.dpeer)
	if err != nil {
		logrus.Errorf("error writing (%v)", err)
		return
	}
	if n != len(data) {
		logrus.Errorf("short write")
		return
	}
	logrus.Infof("transmitted dconn hello attempt")

	for {
		select {
		case <-time.After(1 * time.Second):
			data, err := pb.ToData(pb.NewHello(self.seq.Next(), self.sessn))
			if err != nil {
				logrus.Errorf("error encoding hello (%v)", err)
				return
			}
			n, err := self.dconn.WriteToUDP(data, self.dpeer)
			if err != nil {
				logrus.Errorf("error writing (%v)", err)
				return
			}
			if n != len(data) {
				logrus.Errorf("short write")
				return
			}
			logrus.Infof("transmitted dconn hello attempt")

		case <-closer:
			return
		}
	}
}
