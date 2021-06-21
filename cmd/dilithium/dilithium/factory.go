package dilithium

import (
	"context"
	"crypto/tls"
	"encoding/hex"
	"github.com/lucas-clemente/quic-go"
	"github.com/openziti-incubator/cf"
	"github.com/openziti/dilithium/protocol/westlsworld3"
	"github.com/openziti/dilithium/protocol/westworld3"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net"
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

	case "tls":
		impl := struct{ ProtoProtocol }{}
		impl.listen = func(address string) (Accepter, error) {
			listener, err := tls.Listen("tcp", address, generateTLSConfig())
			if err != nil {
				return nil, errors.Wrap(err, "listen")
			}
			return listener, nil
		}
		impl.dial = func(address string) (net.Conn, error) {
			conn, err := tls.Dial("tcp", address, generateTLSConfig())
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

	case "westworld3":
		p := westworld3.NewBaselineProfile()
		if configPath != "" {
			if configPath != "" {
				data, err := ioutil.ReadFile(configPath)
				if err != nil {
					return nil, errors.Wrapf(err, "unable to read config file [%s]", configPath)
				}
				dataMap := make(map[interface{}]interface{})
				if err = yaml.Unmarshal(data, dataMap); err != nil {
					return nil, errors.Wrapf(err, "unable to unmarshal config data [%s]", configPath)
				}
				if err = p.Load(cf.MapIToMapS(dataMap)); err != nil {
					return nil, errors.Wrapf(err, "unable to load westworld3 profile [%s]", configPath)
				}
			}
		}
		profileId, err := westworld3.AddProfile(p)
		if err != nil {
			return nil, errors.Wrap(err, "unable to register westworld3 profile")
		}
		if configDump {
			logrus.Infof(p.Dump())
		}

		impl := struct{ ProtoProtocol }{}
		impl.listen = func(address string) (Accepter, error) {
			listenAddress, err := net.ResolveUDPAddr("udp", address)
			if err != nil {
				return nil, errors.Wrap(err, "resolve address")
			}
			listener, err := westworld3.Listen(listenAddress, profileId)
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
			conn, err := westworld3.Dial(dialAddress, profileId)
			if err != nil {
				return nil, errors.Wrap(err, "dial")
			}
			return conn, nil
		}
		return impl, nil

	case "westlsworld3":
		p := westworld3.NewBaselineProfile()
		if configPath != "" {
			if configPath != "" {
				data, err := ioutil.ReadFile(configPath)
				if err != nil {
					return nil, errors.Wrapf(err, "unable to read config file [%s]", configPath)
				}
				dataMap := make(map[interface{}]interface{})
				if err = yaml.Unmarshal(data, dataMap); err != nil {
					return nil, errors.Wrapf(err, "unable to unmarshal config data [%s]", configPath)
				}
				if err = p.Load(cf.MapIToMapS(dataMap)); err != nil {
					return nil, errors.Wrapf(err, "unable to load westworld3 profile [%s]", configPath)
				}
			}
		}
		profileId, err := westworld3.AddProfile(p)
		if err != nil {
			return nil, errors.Wrap(err, "unable to register westworld3 profile")
		}
		if configDump {
			logrus.Infof(p.Dump())
		}

		impl := struct{ ProtoProtocol }{}
		impl.listen = func(address string) (Accepter, error) {
			listenAddress, err := net.ResolveUDPAddr("udp", address)
			if err != nil {
				return nil, errors.Wrap(err, "resolve address")
			}
			tlsConfig := generateTLSConfig()
			tlsConfig.ClientAuth = tls.RequireAnyClientCert
			tlsConfig.VerifyConnection = func(cs tls.ConnectionState) error {
				logrus.Infof("negotiated: %s, ciphersuite: %d", cs.NegotiatedProtocol, cs.CipherSuite)
				for _, peerCert := range cs.PeerCertificates {
					logrus.Infof("peer cert = %s", hex.Dump(peerCert.Signature))
				}
				return nil
			}
			listener, err := westlsworld3.Listen(listenAddress, tlsConfig, profileId)
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
			tlsConfig := generateTLSConfig()
			tlsConfig.ClientAuth = tls.RequireAnyClientCert
			tlsConfig.VerifyConnection = func(cs tls.ConnectionState) error {
				logrus.Infof("negotiated: %s, ciphersuite: %d", cs.NegotiatedProtocol, cs.CipherSuite)
				for _, peerCert := range cs.PeerCertificates {
					logrus.Infof("peer cert = %s", hex.Dump(peerCert.Signature))
				}
				return nil
			}
			conn, err := westlsworld3.Dial(dialAddress, tlsConfig, profileId)
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
