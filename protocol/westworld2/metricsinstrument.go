package westworld2

import (
	"encoding/binary"
	"fmt"
	"github.com/emirpasic/gods/trees/btree"
	"github.com/emirpasic/gods/utils"
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
	peers *btree.Tree
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
		peers:               btree.NewWith(1024, utils.UInt32Comparator),
	}
	if err := mi.configure(config); err != nil {
		return nil, err
	}
	go mi.signalHandler()
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
	m := self.metricsForPeer(peer)
	m.errors = append(m.errors, &sample{time.Now(), 1})
}

/* */

/*
 * wire
 */
func (self *metricsInstrument) wireMessageTx(peer *net.UDPAddr, wm *wireMessage) {
	m := self.metricsForPeer(peer)
	m.txBytes = append(m.txBytes, &sample{time.Now(), int64(len(wm.data))})
}

func (self *metricsInstrument) wireMessageRetx(peer *net.UDPAddr, wm *wireMessage) {
	m := self.metricsForPeer(peer)
	m.retxBytes = append(m.retxBytes, &sample{time.Now(), int64(len(wm.data))})
}

func (self *metricsInstrument) wireMessageRx(peer *net.UDPAddr, wm *wireMessage) {
	m := self.metricsForPeer(peer)
	m.rxBytes = append(m.rxBytes, &sample{time.Now(), int64(len(wm.data))})
}

func (self *metricsInstrument) unknownPeer(peer *net.UDPAddr) {
	logrus.Errorf("unknown peer (%s)", peer)
}

func (self *metricsInstrument) readError(peer *net.UDPAddr, err error) {
	logrus.Errorf("read error (%v)", err)
	m := self.metricsForPeer(peer)
	m.errors = append(m.errors, &sample{time.Now(), 1})
}

func (self *metricsInstrument) unexpectedMessageType(peer *net.UDPAddr, mt messageType) {
	logrus.Errorf("unexpected message type (%d)", mt)
	m := self.metricsForPeer(peer)
	m.errors = append(m.errors, &sample{time.Now(), 1})
}

/* */

/*
 * txPortal
 */
func (self *metricsInstrument) txPortalCapacityChanged(peer *net.UDPAddr, capacity int) {
	m := self.metricsForPeer(peer)
	m.txPortalCapacity = append(m.txPortalCapacity, &sample{time.Now(), int64(capacity)})
}

func (self *metricsInstrument) txPortalSzChanged(peer *net.UDPAddr, sz int) {
	/*
	m := self.metricsForPeer(peer)
	m.lock.Lock()
	m.txPortalSz = append(m.txPortalSz, &sample{time.Now(), int64(sz)})
	m.lock.Unlock()
	*/
}

func (self *metricsInstrument) txPortalRxPortalSzChanged(peer *net.UDPAddr, sz int) {
	m := self.metricsForPeer(peer)
	m.txPortalRxPortalSz = append(m.txPortalRxPortalSz, &sample{time.Now(), int64(sz)})
}

func (self *metricsInstrument) newRetxMs(peer *net.UDPAddr, retxMs int) {
	m := self.metricsForPeer(peer)
	m.retxMs = append(m.retxMs, &sample{time.Now(), int64(retxMs)})
}

func (self *metricsInstrument) duplicateAck(peer *net.UDPAddr, _ int32) {
	m := self.metricsForPeer(peer)
	m.duplicateAcks = append(m.duplicateAcks, &sample{time.Now(), 1})
}

/* */

/*
 * rxPortal
 */
func (self *metricsInstrument) rxPortalSzChanged(peer *net.UDPAddr, sz int) {
	/*
	m := self.metricsForPeer(peer)
	m.lock.Lock()
	m.rxPortalSz = append(m.rxPortalSz, &sample{time.Now(), int64(sz)})
	m.lock.Unlock()
	*/
}

func (self *metricsInstrument) duplicateRx(peer *net.UDPAddr, wm *wireMessage) {
	m := self.metricsForPeer(peer)
	m.duplicateRxBytes = append(m.duplicateRxBytes, &sample{time.Now(), int64(len(wm.data))})
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

func (self *metricsInstrument) metricsForPeer(peer *net.UDPAddr) (m *metrics) {
	self.lock.Lock()
	defer self.lock.Unlock()

	pi := netIPtoUint32(peer.IP)
	var ports *btree.Tree
	if v, found := self.peers.Get(pi); found {
		ports = v.(*btree.Tree)
	} else {
		ports = btree.NewWith(1024, utils.IntComparator)
		self.peers.Put(pi, ports)
	}

	if v, found := ports.Get(peer.Port); found {
		m = v.(*metrics)
	} else {
		m = &metrics{lock: new(sync.Mutex)}
		ports.Put(peer.Port, m)
	}

	return
}

func (self *metricsInstrument) writeAllSamples() error {
	self.lock.Lock()
	defer self.lock.Unlock()

	for pi := range self.peers.Keys() {
		v, _ := self.peers.Get(pi)
		ports := v.(*btree.Tree)
		for port := range ports.Keys() {
			v, _ := ports.Get(port)
			m := v.(*metrics)
			peer := uint32ToNetIP(uint32(pi))

			peerName := fmt.Sprintf("%s_%d_", peer.String(), port)
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

func netIPtoUint32(ip net.IP) uint32 {
	if len(ip) == 16 {
		return binary.BigEndian.Uint32(ip[12:16])
	}
	return binary.BigEndian.Uint32(ip)
}


func uint32ToNetIP(nn uint32) net.IP {
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, nn)
	return ip
}