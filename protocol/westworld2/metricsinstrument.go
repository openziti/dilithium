package westworld2

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

type metricsInstrument struct {
	prefix string
	lock   *sync.Mutex

	txBytesQ            chan *peeredSample
	retxBytesQ          chan *peeredSample
	rxBytesQ            chan *peeredSample
	txPortalCapacityQ   chan *peeredSample
	txPortalSzQ         chan *peeredSample
	txPortalRxPortalSzQ chan *peeredSample
	retxMsQ             chan *peeredSample
	duplicateAcksQ      chan *peeredSample
	rxPortalSzQ         chan *peeredSample
	duplicateRxBytesQ   chan *peeredSample
	errorsQ             chan *peeredSample

	peers map[*net.UDPAddr]*metrics
}

type metrics struct {
	lock *sync.Mutex

	txBytes   []*sample
	retxBytes []*sample
	rxBytes   []*sample

	txPortalCapacity   []*sample
	txPortalSz         []*sample
	txPortalRxPortalSz []*sample
	retxMs             []*sample
	duplicateAcks      []*sample

	rxPortalSz       []*sample
	duplicateRxBytes []*sample

	errors []*sample
}

type peeredSample struct {
	peer *net.UDPAddr
	s    *sample
}

type sample struct {
	ts time.Time
	v  int64
}

func newMetricsInstrument(config map[string]interface{}) (Instrument, error) {
	mi := &metricsInstrument{
		lock:                new(sync.Mutex),
		txBytesQ:            make(chan *peeredSample, 10240),
		retxBytesQ:          make(chan *peeredSample, 10240),
		rxBytesQ:            make(chan *peeredSample, 10240),
		txPortalCapacityQ:   make(chan *peeredSample, 10240),
		txPortalSzQ:         make(chan *peeredSample, 10240),
		txPortalRxPortalSzQ: make(chan *peeredSample, 10240),
		retxMsQ:             make(chan *peeredSample, 10240),
		duplicateAcksQ:      make(chan *peeredSample, 10240),
		rxPortalSzQ:         make(chan *peeredSample, 10240),
		duplicateRxBytesQ:   make(chan *peeredSample, 10240),
		errorsQ:             make(chan *peeredSample, 10240),
		peers:               make(map[*net.UDPAddr]*metrics),
	}
	if err := mi.configure(config); err != nil {
		return nil, err
	}
	go mi.signalHandler()
	go mi.manager()
	return mi, nil
}

/*
 * connection
 */
func (self *metricsInstrument) connected(peer *net.UDPAddr) {
}

func (self *metricsInstrument) closed(peer *net.UDPAddr) {
}

func (self *metricsInstrument) connectError(peer *net.UDPAddr, err error) {
	self.errorsQ <- &peeredSample{peer, &sample{time.Now(), 1}}
}

/* */

/*
 * wire
 */
func (self *metricsInstrument) wireMessageTx(peer *net.UDPAddr, wm *wireMessage) {
	self.txBytesQ <- &peeredSample{peer, &sample{time.Now(), int64(len(wm.data))}}
}

func (self *metricsInstrument) wireMessageRetx(peer *net.UDPAddr, wm *wireMessage) {
	self.retxBytesQ <- &peeredSample{peer, &sample{time.Now(), int64(len(wm.data))}}
}

func (self *metricsInstrument) wireMessageRx(peer *net.UDPAddr, wm *wireMessage) {
	self.rxBytesQ <- &peeredSample{peer, &sample{time.Now(), int64(len(wm.data))}}
}

func (self *metricsInstrument) unknownPeer(peer *net.UDPAddr) {
	logrus.Errorf("unknown peer (%s)", peer)
}

func (self *metricsInstrument) readError(peer *net.UDPAddr, err error) {
	logrus.Errorf("read error (%v)", err)
	self.errorsQ <- &peeredSample{peer, &sample{time.Now(), 1}}
}

func (self *metricsInstrument) unexpectedMessageType(peer *net.UDPAddr, mt messageType) {
	logrus.Errorf("unexpected message type (%d)", mt)
	self.errorsQ <- &peeredSample{peer, &sample{time.Now(), 1}}
}

/* */

/*
 * txPortal
 */
func (self *metricsInstrument) txPortalCapacityChanged(peer *net.UDPAddr, capacity int) {
	self.txPortalCapacityQ <- &peeredSample{peer, &sample{time.Now(), int64(capacity)}}
}

func (self *metricsInstrument) txPortalSzChanged(peer *net.UDPAddr, sz int) {
	self.txPortalSzQ <- &peeredSample{peer, &sample{time.Now(), int64(sz)}}
}

func (self *metricsInstrument) txPortalRxPortalSzChanged(peer *net.UDPAddr, sz int) {
	self.txPortalRxPortalSzQ <- &peeredSample{peer, &sample{time.Now(), int64(sz)}}
}

func (self *metricsInstrument) newRetxMs(peer *net.UDPAddr, retxMs int) {
	self.retxMsQ <- &peeredSample{peer, &sample{time.Now(), int64(retxMs)}}
}

func (self *metricsInstrument) duplicateAck(peer *net.UDPAddr, _ int32) {
	self.duplicateAcksQ <- &peeredSample{peer, &sample{time.Now(), 1}}
}

/* */

/*
 * rxPortal
 */
func (self *metricsInstrument) rxPortalSzChanged(peer *net.UDPAddr, sz int) {
	self.rxPortalSzQ <- &peeredSample{peer, &sample{time.Now(), int64(sz)}}
}

func (self *metricsInstrument) duplicateRx(peer *net.UDPAddr, wm *wireMessage) {
	self.duplicateRxBytesQ <- &peeredSample{peer, &sample{time.Now(), int64(len(wm.data))}}
}

/* */

/*
 * allocation
 */
func (self *metricsInstrument) allocate(_ string) {
}

/* */

func (self *metricsInstrument) configure(data map[string]interface{}) error {
	if v, found := data["prefix"]; found {
		if prefix, ok := v.(string); ok {
			self.prefix = prefix
			logrus.Infof("writing metrics data to prefix [%s]", self.prefix)
		} else {
			return errors.New("invalid 'prefix' type")
		}
	}
	return nil
}

func (self *metricsInstrument) manager() {
	for {
		select {
		case ps, ok := <-self.txBytesQ:
			if !ok {
				return
			}
			m := self.metricsForPeer(ps.peer)
			m.txBytes = append(m.txBytes, ps.s)

		case ps, ok := <-self.retxBytesQ:
			if !ok {
				return
			}
			m := self.metricsForPeer(ps.peer)
			m.retxBytes = append(m.retxBytes, ps.s)

		case ps, ok := <-self.rxBytesQ:
			if !ok {
				return
			}
			m := self.metricsForPeer(ps.peer)
			m.rxBytes = append(m.rxBytes, ps.s)

		case ps, ok := <-self.txPortalCapacityQ:
			if !ok {
				return
			}
			m := self.metricsForPeer(ps.peer)
			m.txPortalCapacity = append(m.txPortalCapacity, ps.s)

		case ps, ok := <-self.txPortalSzQ:
			if !ok {
				return
			}
			m := self.metricsForPeer(ps.peer)
			m.txPortalSz = append(m.txPortalSz, ps.s)

		case ps, ok := <-self.txPortalRxPortalSzQ:
			if !ok {
				return
			}
			m := self.metricsForPeer(ps.peer)
			m.txPortalRxPortalSz = append(m.txPortalRxPortalSz, ps.s)

		case ps, ok := <-self.retxMsQ:
			if !ok {
				return
			}
			m := self.metricsForPeer(ps.peer)
			m.retxMs = append(m.retxMs, ps.s)

		case ps, ok := <-self.duplicateAcksQ:
			if !ok {
				return
			}
			m := self.metricsForPeer(ps.peer)
			m.duplicateAcks = append(m.duplicateAcks, ps.s)

		case ps, ok := <-self.rxPortalSzQ:
			if !ok {
				return
			}
			m := self.metricsForPeer(ps.peer)
			m.rxPortalSz = append(m.rxPortalSz, ps.s)

		case ps, ok := <-self.duplicateRxBytesQ:
			if !ok {
				return
			}
			m := self.metricsForPeer(ps.peer)
			m.duplicateRxBytes = append(m.duplicateRxBytes, ps.s)

		case ps, ok := <-self.errorsQ:
			if !ok {
				return
			}
			m := self.metricsForPeer(ps.peer)
			m.errors = append(m.errors, ps.s)
		}
	}
}

func (self *metricsInstrument) metricsForPeer(peer *net.UDPAddr) (m *metrics) {
	if pm, f := self.peers[peer]; f {
		m = pm
	} else {
		m = &metrics{}
		self.peers[peer] = m
	}
	return
}

func (self *metricsInstrument) writeAllSamples() error {
	self.lock.Lock()
	defer self.lock.Unlock()

	for peer, m := range self.peers {
		peerName := fmt.Sprintf("%s_%d_", peer.IP.String(), peer.Port)
		if err := os.MkdirAll(self.prefix, os.ModePerm); err != nil {
			return err
		}
		outpath, err := ioutil.TempDir(self.prefix, peerName)
		if err != nil {
			return err
		}
		logrus.Infof("writing metrics to: %s", outpath)

		if err := self.writeSamples("txBytes", outpath, m.txBytes); err != nil {
			return err
		}
		m.txBytes = nil
		if err := self.writeSamples("retxBytes", outpath, m.retxBytes); err != nil {
			return err
		}
		m.retxBytes = nil
		if err := self.writeSamples("rxBytes", outpath, m.rxBytes); err != nil {
			return err
		}
		m.rxBytes = nil
		if err := self.writeSamples("txPortalCapacity", outpath, m.txPortalCapacity); err != nil {
			return err
		}
		m.txPortalCapacity = nil
		if err := self.writeSamples("txPortalSz", outpath, m.txPortalSz); err != nil {
			return err
		}
		m.txPortalSz = nil
		if err := self.writeSamples("txPortalRxPortalSz", outpath, m.txPortalRxPortalSz); err != nil {
			return err
		}
		m.txPortalRxPortalSz = nil
		if err := self.writeSamples("retxMs", outpath, m.retxMs); err != nil {
			return err
		}
		m.retxMs = nil
		if err := self.writeSamples("duplicateAcks", outpath, m.duplicateAcks); err != nil {
			return err
		}
		m.duplicateAcks = nil
		if err := self.writeSamples("rxPortalSz", outpath, m.rxPortalSz); err != nil {
			return err
		}
		m.rxPortalSz = nil
		if err := self.writeSamples("duplicateRxBytes", outpath, m.duplicateRxBytes); err != nil {
			return err
		}
		m.duplicateRxBytes = nil
		if err := self.writeSamples("errors", outpath, m.errors); err != nil {
			return err
		}
		m.errors = nil
	}
	return nil
}

func (self *metricsInstrument) writeSamples(name, outPath string, samples []*sample) error {
	path := filepath.Join(outPath, fmt.Sprintf("%s.csv", name))
	oF, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return err
	}
	defer func() {
		_ = oF.Close()
	}()
	for _, sample := range samples {
		line := fmt.Sprintf("%d,%d\n", sample.ts.UnixNano(), sample.v)
		n, err := oF.Write([]byte(line))
		if err != nil {
			return err
		}
		if n != len(line) {
			return errors.New("short write")
		}
	}
	logrus.Infof("wrote [%d] samples to [%s]", len(samples), path)
	return nil
}

func (self *metricsInstrument) signalHandler() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGUSR2)

	for {
		s := <-c
		if s == syscall.SIGUSR2 {
			if err := self.writeAllSamples(); err != nil {
				logrus.Errorf("error writing samples (%v)", err)
			}
		}
	}
}
