package dilithium

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"github.com/lucas-clemente/quic-go"
	"github.com/michaelquigley/dilithium/protocol/conduit"
	"github.com/michaelquigley/dilithium/protocol/westworld"
	"github.com/michaelquigley/dilithium/protocol/westworld2"
	"github.com/pkg/errors"
	"math/big"
	"net"
	"time"
)

type Protocol interface {
	Listen(address string) (Accepter, error)
	Dial(address string) (net.Conn, error)
}

type ProtoProtocol struct {
	listen func(address string) (Accepter, error)
	dial   func(address string) (net.Conn, error)
}

type Accepter interface {
	Accept() (net.Conn, error)
}

func (self ProtoProtocol) Listen(address string) (Accepter, error) { return self.listen(address) }
func (self ProtoProtocol) Dial(address string) (net.Conn, error)   { return self.dial(address) }

func ProtocolFor(protocol string) (Protocol, error) {
	switch protocol {
	case "tcp":
		impl := struct{ ProtoProtocol }{}
		impl.listen = func(address string) (Accepter, error) {
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

	case "quic":
		impl := struct{ ProtoProtocol }{}
		impl.listen = func(address string) (Accepter, error) {
			listener, err := quic.ListenAddr(address, generateTLSConfig(), nil)
			if err != nil {
				return nil, errors.Wrap(err, "listen")
			}

			return &quicAccepter{listener}, nil
		}
		impl.dial = func(address string) (net.Conn, error) {
			session, err := quic.DialAddr(address, generateTLSConfig(), nil)
			if err != nil {
				return nil, errors.Wrap(err, "dial")
			}

			stream, err := session.OpenStreamSync(context.Background())
			if err != nil {
				return nil, errors.Wrap(err, "stream")
			}

			return &quicConn{session, stream}, nil
		}

		return impl, nil

	case "conduit":
		impl := struct{ ProtoProtocol }{}
		impl.listen = func(address string) (Accepter, error) {
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

	case "westworld":
		impl := struct{ ProtoProtocol }{}
		impl.listen = func(address string) (Accepter, error) {
			listenAddress, err := net.ResolveUDPAddr("udp", address)
			if err != nil {
				return nil, errors.Wrap(err, "resolve address")
			}

			listener, err := westworld.Listen(listenAddress)
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

			conn, err := westworld.Dial(dialAddress)
			if err != nil {
				return nil, errors.Wrap(err, "dial")
			}

			return conn, nil
		}

		return impl, nil

	case "westworld2":
		impl := struct{ ProtoProtocol }{}
		impl.listen = func(address string) (Accepter, error) {
			listenAddress, err := net.ResolveUDPAddr("udp", address)
			if err != nil {
				return nil, errors.Wrap(err, "resolve address")
			}

			listener, err := westworld2.Listen(listenAddress)
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

			conn, err := westworld2.Dial(dialAddress)
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

func generateTLSConfig() *tls.Config {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		panic(err)
	}
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotBefore:    time.Now().Add(-24 * time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1)},
		IsCA:         true,
	}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		panic(err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		panic(err)
	}
	return &tls.Config{
		Certificates:       []tls.Certificate{tlsCert},
		NextProtos:         []string{"dilithium"},
		InsecureSkipVerify: true,
	}
}
