package conduit

import (
	"github.com/pkg/errors"
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
	m, ok := <-self.readQueue
	if ok {
		n := copy(p, m.payload)
		return n, nil
	}
	return 0, errors.New("closed")
}

func (self *listenerConn) Write(p []byte) (n int, err error) {
	data, err := newPayloadMessage(self.sequence.next(), p).marshal()
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
	m, _, err := readMessage(self.conn)
	if err != nil {
		return 0, errors.Wrap(err, "read message")
	}
	if m.message == Payload {
		n := copy(p, m.payload)
		return n, nil
	} else {
		return 0, errors.New("invalid message")
	}
}

func (self *dialerConn) Write(p []byte) (n int, err error) {
	data, err := newPayloadMessage(self.sequence.next(), p).marshal()
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
