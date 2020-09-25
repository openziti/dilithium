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
	prefix    string
	lock      *sync.Mutex
	instances []*metricsInstrumentInstance
}

func newMetricsInstrument(config map[string]interface{}) (Instrument, error) {
	mi := &metricsInstrument{lock: new(sync.Mutex)}
	if err := mi.configure(config); err != nil {
		return nil, err
	}
	go mi.signalHandler()
	return mi, nil
}

func (self *metricsInstrument) newInstance(peer *net.UDPAddr) InstrumentInstance {
	self.lock.Lock()
	defer self.lock.Unlock()
	mi := &metricsInstrumentInstance{peer: peer, lock: new(sync.Mutex)}
	self.instances = append(self.instances, mi)
	return mi
}

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

func (self *metricsInstrument) writeAllSamples() error {
	self.lock.Lock()
	defer self.lock.Unlock()

	for _, mi := range self.instances {
		peerName := fmt.Sprintf("%s_", mi.peer.String())
		if err := os.MkdirAll(self.prefix, os.ModePerm); err != nil {
			return err
		}
		outpath, err := ioutil.TempDir(self.prefix, peerName)
		if err != nil {
			return err
		}
		logrus.Infof("writing metrics to: %s", outpath)

		if err := self.writeSamples("txBytes", outpath, mi.txBytes); err != nil {
			return err
		}
		mi.txBytes = nil
		if err := self.writeSamples("retxBytes", outpath, mi.retxBytes); err != nil {
			return err
		}
		mi.retxBytes = nil
		if err := self.writeSamples("rxBytes", outpath, mi.rxBytes); err != nil {
			return err
		}
		mi.rxBytes = nil
		if err := self.writeSamples("txPortalCapacity", outpath, mi.txPortalCapacity); err != nil {
			return err
		}
		mi.txPortalCapacity = nil
		if err := self.writeSamples("txPortalSz", outpath, mi.txPortalSz); err != nil {
			return err
		}
		mi.txPortalSz = nil
		if err := self.writeSamples("txPortalRxPortalSz", outpath, mi.txPortalRxPortalSz); err != nil {
			return err
		}
		mi.txPortalRxPortalSz = nil
		if err := self.writeSamples("retxMs", outpath, mi.retxMs); err != nil {
			return err
		}
		mi.retxMs = nil
		if err := self.writeSamples("duplicateAcks", outpath, mi.duplicateAcks); err != nil {
			return err
		}
		mi.duplicateAcks = nil
		if err := self.writeSamples("rxPortalSz", outpath, mi.rxPortalSz); err != nil {
			return err
		}
		mi.rxPortalSz = nil
		if err := self.writeSamples("duplicateRxBytes", outpath, mi.duplicateRxBytes); err != nil {
			return err
		}
		mi.duplicateRxBytes = nil
		if err := self.writeSamples("errors", outpath, mi.errors); err != nil {
			return err
		}
		mi.errors = nil
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

type metricsInstrumentInstance struct {
	lock *sync.Mutex
	peer *net.UDPAddr

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

	allocations []*sample
	errors      []*sample
}

type sample struct {
	ts time.Time
	v  int64
}

/*
 * connection
 */
func (self *metricsInstrumentInstance) connected(_ *net.UDPAddr) {
}

func (self *metricsInstrumentInstance) closed(_ *net.UDPAddr) {
}

func (self *metricsInstrumentInstance) connectError(_ *net.UDPAddr, err error) {
	logrus.Errorf("connect error (%v)", err)
	self.lock.Lock()
	defer self.lock.Unlock()
	self.errors = append(self.errors, &sample{time.Now(), 1})
}
/* */

/*
 * wire
 */
func (self *metricsInstrumentInstance) wireMessageTx(peer *net.UDPAddr, wm *wireMessage) {
	self.lock.Lock()
	defer self.lock.Unlock()
	self.txBytes = append(self.txBytes, &sample{time.Now(), int64(len(wm.data))})
}

func (self *metricsInstrumentInstance) wireMessageRetx(peer *net.UDPAddr, wm *wireMessage) {
	self.lock.Lock()
	defer self.lock.Unlock()
	self.retxBytes = append(self.retxBytes, &sample{time.Now(), int64(len(wm.data))})
}

func (self *metricsInstrumentInstance) wireMessageRx(peer *net.UDPAddr, wm *wireMessage) {
	self.lock.Lock()
	defer self.lock.Unlock()
	self.rxBytes = append(self.rxBytes, &sample{time.Now(), int64(len(wm.data))})
}

func (self *metricsInstrumentInstance) unknownPeer(peer *net.UDPAddr) {
	logrus.Errorf("unknown peer (%s)", peer)
	self.lock.Lock()
	defer self.lock.Unlock()
	self.errors = append(self.errors, &sample{time.Now(), 1})
}

func (self *metricsInstrumentInstance) readError(peer *net.UDPAddr, err error) {
	logrus.Errorf("read error (%v)", err)
	self.lock.Lock()
	defer self.lock.Unlock()
	self.errors = append(self.errors, &sample{time.Now(), 1})
}

func (self *metricsInstrumentInstance) unexpectedMessageType(peer *net.UDPAddr, mt messageType) {
	logrus.Errorf("unexpected message type (%d)", mt)
	self.lock.Lock()
	defer self.lock.Unlock()
	self.errors = append(self.errors, &sample{time.Now(), 1})
}
/* */

/*
 * txPortal
 */
func (self *metricsInstrumentInstance) txPortalCapacityChanged(peer *net.UDPAddr, capacity int) {
	self.lock.Lock()
	defer self.lock.Unlock()
	self.txPortalCapacity = append(self.txPortalCapacity, &sample{time.Now(), int64(capacity)})
}

func (self *metricsInstrumentInstance) txPortalSzChanged(peer *net.UDPAddr, sz int) {
	self.lock.Lock()
	defer self.lock.Unlock()
	self.txPortalSz = append(self.txPortalSz, &sample{time.Now(), int64(sz)})
}

func (self *metricsInstrumentInstance) txPortalRxPortalSzChanged(peer *net.UDPAddr, sz int) {
	self.lock.Lock()
	defer self.lock.Unlock()
	self.txPortalRxPortalSz = append(self.txPortalRxPortalSz, &sample{time.Now(), int64(sz)})
}

func (self *metricsInstrumentInstance) newRetxMs(peer *net.UDPAddr, retxMs int) {
	self.lock.Lock()
	defer self.lock.Unlock()
	self.retxMs = append(self.retxMs, &sample{time.Now(), int64(retxMs)})
}

func (self *metricsInstrumentInstance) duplicateAck(peer *net.UDPAddr, _ int32) {
	self.lock.Lock()
	defer self.lock.Unlock()
	self.duplicateAcks = append(self.duplicateAcks, &sample{time.Now(), 1})
}
/* */

/*
 * rxPortal
 */
func (self *metricsInstrumentInstance) rxPortalSzChanged(peer *net.UDPAddr, sz int) {
	self.lock.Lock()
	defer self.lock.Unlock()
	self.rxPortalSz = append(self.rxPortalSz, &sample{time.Now(), int64(sz)})
}

func (self *metricsInstrumentInstance) duplicateRx(peer *net.UDPAddr, wm *wireMessage) {
	self.lock.Lock()
	defer self.lock.Unlock()
	self.duplicateRxBytes = append(self.duplicateRxBytes, &sample{time.Now(), int64(len(wm.data))})
}
/* */

/*
 * allocation
 */
func (self *metricsInstrumentInstance) allocate(_ string) {
	self.lock.Lock()
	defer self.lock.Unlock()
	self.allocations = append(self.allocations, &sample{time.Now(), 1})
}
/* */
