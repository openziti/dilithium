package blaster

import (
	"github.com/michaelquigley/dilithium/blaster/pb"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"time"
)

func (self *listenerConn) hello() error {
	/*
	 * Rx cConn Sync
	 */
	logrus.Infof("start rx cConn sync")
	wm, err := pb.ReadMessage(self.cConn)
	if err != nil {
		return errors.Wrap(err, "read sync")
	}
	if wm.Type != pb.MessageType_SYNC {
		return errors.Errorf("expected sync got mt [%d]", wm.Type)
	}
	/* */

	/*
	 * Tx cConn Hello
	 */
	logrus.Infof("start tx cConn hello")
	err = pb.WriteMessage(pb.NewHello(self.cSeq.Next(), self.session), self.cConn)
	if err != nil {
		return errors.Wrap(err, "write hello")
	}
	/* */

	/*
	 * Rx dConn Hello
	 */
	logrus.Infof("start rx dConn hello")
	start := time.Now()
	success := false
	done := false
	for !done {
		if time.Now().Sub(start).Seconds() >= 5 {
			break // hello timeout
		}

		select {
		case wmp := <-self.dRxQueue:
			if wmp.WireMessage.Type == pb.MessageType_HELLO {
				if wmp.WireMessage.HelloPayload.Session == self.session {
					self.dPeer = wmp.FromPeer
					self.cListener.connected[self.dPeer.String()] = self
					delete(self.cListener.syncing, self.session)
					success = true
					done = true
					break
				}
			} else {
				logrus.Errorf("unexpected message type [%d]", wmp.WireMessage.Type)
			}

		case <-time.After(5 * time.Second):
			done = true
			break // packet timeout
		}
	}
	if success {
		err = pb.WriteMessage(pb.NewOk(self.cSeq.Next()), self.cConn)
		if err != nil {
			return errors.Wrap(err, "ok write")
		}
	} else {
		return errors.New("dConn hello timeout")
	}
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

	wmp, err := pb.ReadMessage(self.cConn)
	if err != nil {
		return errors.Wrap(err, "ok decode")
	}
	if wmp.Type != pb.MessageType_OK {
		return errors.New("not ok")
	}
	if err := self.cConn.SetReadDeadline(time.Time{}); err != nil {
		return errors.Wrap(err, "clear cConn deadline")
	}
	/* */

	logrus.Infof("connection established")

	return nil
}

func (self *listenerConn) helloTxDConn(closer chan struct{}) {
	logrus.Infof("started")
	defer logrus.Warnf("exited")

	data, err := pb.ToData(pb.NewHello(self.dSeq.Next(), self.session))
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
	logrus.Infof("transmitted dConn hello attempt")

	for {
		select {
		case <-time.After(1 * time.Second):
			data, err := pb.ToData(pb.NewHello(self.dSeq.Next(), self.session))
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
			logrus.Infof("transmitted dConn hello attempt")

		case <-closer:
			return
		}
	}
}
