package conduit

import (
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"net"
	"time"
)

type listenerConn struct {
	conn      *net.UDPConn
	local     *net.UDPAddr
	peer      *net.UDPAddr
	readQueue chan *message
	sequence  *sequence
	rxWindow  *rxWindow
	txWindow  *txWindow
}

func newListenerConn(conn *net.UDPConn, local *net.UDPAddr, peer *net.UDPAddr) *listenerConn {
	return &listenerConn{
		conn:      conn,
		local:     local,
		peer:      peer,
		readQueue: make(chan *message, 1024),
		sequence:  newSequence(),
		rxWindow:  newRxWindow(conn, peer),
		txWindow:  newTxWindow(conn, peer),
	}
}

func (self *listenerConn) Read(p []byte) (n int, err error) {
	n, err = self.rxWindow.read(p)
	if n > 0 {
		return
	}

	for {
		m, ok := <-self.readQueue
		if ok {
			if m.message == Payload {
				logrus.Infof("[#%d](%d) <-", m.sequence, len(m.payload))
				self.rxWindow.rx(m)
				n, err = self.rxWindow.read(p)
				if err == nil {
					logrus.Infof("[](%d) <-", n)
				}
				return

			} else if m.message == Ack {
				if sequence, err := readInt32(m.payload); err == nil {
					logrus.Infof("[@%d] <-", sequence)
					self.txWindow.ack(sequence)
				} else {
					return 0, errors.New("invalid ack")
				}

			} else {
				return 0, errors.New("invalid message")
			}
		}

		return 0, errors.New("closed")
	}
}

func (self *listenerConn) Write(p []byte) (n int, err error) {
	m := newPayloadMessage(self.sequence.next(), p)
	self.txWindow.tx(m)

	data, err := m.marshal()
	if err != nil {
		return 0, errors.Wrap(err, "marshal")
	}
	n, err = self.conn.WriteTo(data, self.peer)
	if err != nil {
		return
	}
	if n != len(data) {
		return 0, errors.New("short write")
	}
	logrus.Infof("[#%d](%d) ->", m.sequence, len(m.payload))
	return len(p), nil
}

func (self *listenerConn) Close() error {
	return self.conn.Close()
}

func (self *listenerConn) RemoteAddr() net.Addr {
	return self.peer
}

func (self *listenerConn) LocalAddr() net.Addr {
	return self.conn.LocalAddr()
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

func (self *listenerConn) queue(m *message) {
	self.readQueue <- m
}

type dialerConn struct {
	conn     *net.UDPConn
	local    *net.UDPAddr
	peer     *net.UDPAddr
	sequence *sequence
	rxWindow *rxWindow
	txWindow *txWindow
}

func newDialerConn(conn *net.UDPConn, local *net.UDPAddr, peer *net.UDPAddr) *dialerConn {
	return &dialerConn{
		conn:     conn,
		local:    local,
		peer:     peer,
		sequence: newSequence(),
		rxWindow: newRxWindow(conn, peer),
		txWindow: newTxWindow(conn, peer),
	}
}

func (self *dialerConn) Read(p []byte) (n int, err error) {
	n, err = self.rxWindow.read(p)
	if n > 0 {
		return
	}

	for {
		m, _, err := readMessage(self.conn)
		if err != nil {
			return 0, errors.Wrap(err, "read message")
		}

		if m.message == Payload {
			logrus.Infof("[#%d](%d) <-", m.sequence, len(m.payload))
			self.rxWindow.rx(m)
			n, err = self.rxWindow.read(p)
			if err == nil {
				logrus.Infof("[](%d) <-", n)
			}
			return n, err

		} else if m.message == Ack {
			if sequence, err := readInt32(m.payload); err == nil {
				logrus.Infof("[@%d] <-", sequence)
				self.txWindow.ack(sequence)
			} else {
				return 0, errors.New("invalid ack")
			}
		} else {
			return 0, errors.New("invalid message")
		}
	}
}

func (self *dialerConn) Write(p []byte) (n int, err error) {
	m := newPayloadMessage(self.sequence.next(), p)
	self.txWindow.tx(m)

	data, err := m.marshal()
	if err != nil {
		return 0, errors.Wrap(err, "marshal")
	}
	n, err = self.conn.WriteTo(data, self.peer)
	if err != nil {
		return
	}
	if n != len(data) {
		return 0, errors.New("short write")
	}
	logrus.Infof("[#%d](%d) ->", m.sequence, len(m.payload))
	return len(p), nil
}

func (self *dialerConn) Close() error {
	return self.conn.Close()
}

func (self *dialerConn) RemoteAddr() net.Addr {
	return self.peer
}

func (self *dialerConn) LocalAddr() net.Addr {
	return self.conn.LocalAddr()
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
