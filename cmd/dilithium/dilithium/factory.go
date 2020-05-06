package dilithium

import (
	"github.com/michaelquigley/dilithium/conduit"
	"github.com/pkg/errors"
	"net"
)

type Protocol interface {
	Listen(address string) (net.Listener, error)
	Dial(address string) (net.Conn, error)
}

type ProtoProtocol struct {
	listen func(address string) (net.Listener, error)
	dial func(address string) (net.Conn, error)
}

func (self ProtoProtocol) Listen(address string) (net.Listener, error) { return self.listen(address) }
func (self ProtoProtocol) Dial(address string) (net.Conn, error) { return self.dial(address) }

func ProtocolFor(protocol string) (Protocol, error) {
	switch protocol {
	case "tcp":
		impl := struct{ ProtoProtocol }{}
		impl.listen = func(address string) (net.Listener, error) {
			listenAddress, err := net.ResolveTCPAddr("tcp", address)
			if err != nil {
				return nil, errors.Wrap(err, "resolve address")
			}

			listener, err := net.ListenTCP("tcp", listenAddress)
			if err != nil {
				return nil, errors.Wrap(err, "listen")
			}

			return listener, nil
		}
		impl.dial = func(address string) (net.Conn, error) {
			dialAddress, err := net.ResolveTCPAddr("tcp", address)
			if err != nil {
				return nil, errors.Wrap(err, "resolve address")
			}

			conn, err := net.DialTCP("tcp", nil, dialAddress)
			if err != nil {
				return nil, errors.Wrap(err, "dial")
			}

			return conn, nil
		}
		return impl, nil

	case "conduit":
		impl := struct{ ProtoProtocol }{}
		impl.listen = func(address string) (net.Listener, error) {
			listenAddress, err := net.ResolveUDPAddr("udp", address)
			if err != nil {
				return nil, errors.Wrap(err, "resolve address")
			}

			listener, err := conduit.Listen(listenAddress)
			if err != nil {
				return nil, errors.Wrap(err, "listen")
			}

			return listener, nil
		}
		impl.dial = func(address string) (net.Conn, error) {
			dialAddress, err := net.ResolveUDPAddr("udp", address)
			if err != nil {
				return nil, errors.Wrap(err, "resolve address")
			}

			conn, err := conduit.Dial(dialAddress)
			if err != nil {
				return nil, errors.Wrap(err, "dial")
			}

			return conn, nil
		}
		return impl, nil

	default:
		return nil, errors.Errorf("unsupported protocol [%s]", protocol)
	}
}