package dilithium

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"github.com/lucas-clemente/quic-go"
	"github.com/openziti/dilithium/cf"
	"github.com/openziti/dilithium/protocol/westworld2"
	"github.com/openziti/dilithium/protocol/westworld3"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"io/ioutil"
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

	case "westworld2":
		cfg := westworld2.NewDefaultConfig()
		if configPath != "" {
			data, err := ioutil.ReadFile(configPath)
			if err != nil {
				return nil, errors.Wrapf(err, "unable to read config file [%s]", configPath)
			}
			dataMap := make(map[interface{}]interface{})
			if err = yaml.Unmarshal(data, dataMap); err != nil {
				return nil, errors.Wrapf(err, "unable to unmarshal config data [%s]", configPath)
			}
			if err = cfg.Load(dataMap); err != nil {
				return nil, errors.Wrapf(err, "unable to load westworld2 config [%s]", configPath)
			}
		}
		if configDump {
			logrus.Infof(cfg.Dump())
		}

		impl := struct{ ProtoProtocol }{}
		impl.listen = func(address string) (Accepter, error) {
			listenAddress, err := net.ResolveUDPAddr("udp", address)
			if err != nil {
				return nil, errors.Wrap(err, "resolve address")
			}
			listener, err := westworld2.Listen(listenAddress, cfg)
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
			conn, err := westworld2.Dial(dialAddress, cfg)
			if err != nil {
				return nil, errors.Wrap(err, "dial")
			}
			return conn, nil
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
