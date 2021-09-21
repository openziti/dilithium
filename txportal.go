package dilithium

import (
	"github.com/emirpasic/gods/trees/btree"
	"github.com/emirpasic/gods/utils"
	"github.com/openziti/dilithium/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"io"
	"math"
	"sync"
	"time"
)

// TxPortal manages the outgoing data transmitted by a communication instance. It is one half of a TxPortal->RxPortal
// communication pair. TxPortal is primarily concerned with optimizing the transmission rate over lossy Adapter
// implementations, while ensuring reliability.
//
type TxPortal struct {
	lock      *sync.Mutex
	tree      *btree.Tree
	lastTx    time.Time
	adapter   Adapter
	alg       TxAlgorithm
	monitor   *TxMonitor
	closer    *Closer
	closeSent bool
	closed    bool
	pool      *Pool
}

func NewTxPortal(adapter Adapter, alg TxAlgorithm, closer *Closer) *TxPortal {
	txp := &TxPortal{
		lock:    new(sync.Mutex),
		tree:    btree.NewWith(alg.Profile().MaxTreeSize, utils.Int32Comparator),
		adapter: adapter,
		alg:     alg,
		closer:  closer,
		pool:    alg.Profile().NewPool("tx"),
	}
	txp.alg.SetLock(txp.lock)
	txp.monitor = newTxMonitor(txp.lock, txp.alg, txp.adapter)
	txp.monitor.setRetxCallback(func(size int) {
		txp.alg.Retransmission(size)
	})
	return txp
}

func (txp *TxPortal) Start() {
	txp.monitor.start()
	if txp.alg.Profile().SendKeepalive {
		go txp.keepaliveSender()
	}
}

func (txp *TxPortal) Tx(p []byte, seq *util.Sequence) (n int, err error) {
	txp.lock.Lock()
	defer txp.lock.Unlock()

	if txp.closed {
		return -1, io.EOF
	}

	remaining := len(p)
	n = 0
	for remaining > 0 {
		segmentSize := int(math.Min(float64(remaining), float64(txp.alg.Profile().MaxSegmentSize)))

		txp.alg.Tx(segmentSize)

		var rtt *uint16
		if txp.alg.ProbeRTT() {
			now := time.Now()
			rtt = new(uint16)
			*rtt = uint16(now.UnixNano() / int64(time.Millisecond))
			segmentSize -= 2
		}

		wm, err := newData(seq.Next(), rtt, p[n:n+segmentSize], txp.pool)
		if err != nil {
			return 0, errors.Wrap(err, "new data")
		}
		txp.tree.Put(wm.Seq, wm)

		if err := writeWireMessage(wm, txp.adapter); err != nil {
			return 0, errors.Wrap(err, "tx")
		}
		txp.lastTx = time.Now()

		txp.monitor.add(wm)

		n += segmentSize
		remaining -= segmentSize
	}

	return n, nil
}

func (txp *TxPortal) ack(acks []Ack) error {
	txp.lock.Lock()
	defer txp.lock.Unlock()

	for _, ack := range acks {
		for seq := ack.Start; seq <= ack.End; seq++ {
			if v, found := txp.tree.Get(seq); found {
				wm := v.(*WireMessage)
				txp.monitor.remove(wm)
				txp.tree.Remove(seq)

				switch wm.messageType() {
				case DATA:
					size, err := wm.asDataSize()
					if err != nil {
						return errors.Wrap(err, "internal tree error")
					}
					txp.alg.Success(int(size))

				case CLOSE:
					//

				default:
					logrus.Warnf("acked suspicious message type in tree [%d]", wm.messageType())
				}
				wm.buf.Unref()

			} else {
				txp.alg.DuplicateAck()
			}
		}
	}
	return nil
}

func (txp *TxPortal) sendClose(seq *util.Sequence) error {
	txp.lock.Lock()
	defer txp.lock.Unlock()

	if !txp.closeSent {
		wm, err := newClose(seq.Next(), txp.pool)
		if err != nil {
			return errors.Wrap(err, "close")
		}
		txp.tree.Put(wm.Seq, wm)
		txp.monitor.add(wm)

		if err := writeWireMessage(wm, txp.adapter); err != nil {
			return errors.Wrap(err, "tx close")
		}
		txp.closer.txCloseSeqIn <- wm.Seq
		txp.closeSent = true
	}

	return nil
}

func (txp *TxPortal) close() {
	txp.closed = true
	txp.monitor.close()
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
				if err := writeWireMessage(keepalive, txp.adapter); err == nil {
					txp.lastTx = time.Now()
				} else {
					logrus.Errorf("error sending keepalive (%v)", err)
				}
			}
		}
	}
}
