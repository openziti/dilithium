package dilithium

import (
	"github.com/emirpasic/gods/trees/btree"
	"github.com/emirpasic/gods/utils"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

// TxPortal manages the outgoing data transmitted by a communication instance. It is one half of a TxPortal->RxPortal
// communication pair. TxPortal is primarily concerned with optimizing the transmission rate over lossy Transport
// implementations, while ensuring reliability.
//
type TxPortal struct {
	lock      *sync.Mutex
	tree      *btree.Tree
	lastTx    time.Time
	transport Transport
	alg       TxAlgorithm
	monitor   *TxMonitor
	closer    *Closer
	closed    bool
	pool      *Pool
}

func newTxPortal(transport Transport, alg TxAlgorithm, closer *Closer, pool *Pool) *TxPortal {
	txp := &TxPortal{
		lock:      new(sync.Mutex),
		tree:      btree.NewWith(alg.Profile().MaxTreeSize, utils.Int32Comparator),
		transport: transport,
		alg:       alg,
		closer:    closer,
	}
	txp.monitor = newTxMonitor(txp.lock, txp.alg, txp.transport)
	//txp.monitor.setRetxCallback()
	return txp
}

func (txp *TxPortal) start() {
	txp.monitor.start()
	if txp.alg.Profile().SendKeepalive {
		go txp.keepaliveSender()
	}
}

func (txp *TxPortal) keepaliveSender() {
	logrus.Info("started")
	defer logrus.Info("exited")

	for {
		time.Sleep(1 * time.Second)
		if txp.closed {
			return
		}
		if time.Since(txp.lastTx).Milliseconds() >= txp.alg.Profile().ConnectionTimeout.Milliseconds()/2 {
			if keepalive, err := newKeepalive(txp.alg.RxPortalSize(), txp.pool); err == nil {
				if err := writeWireMessage(keepalive, txp.transport); err == nil {
					txp.lastTx = time.Now()
				} else {
					logrus.Errorf("error sending keepalive (%v)", err)
				}
			}
		}
	}
}
