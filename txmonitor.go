package dilithium

import (
	"github.com/openziti/dilithium/util"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

// TxMonitor is responsible for managing in-flight payloads, retransmitting payloads when their timeout expires.
//
type TxMonitor struct {
	lock         *sync.Mutex
	ready        *sync.Cond
	alg          TxAlgorithm
	transport    Transport
	waitlist     waitlist
	closed       bool
	retxCallback func()
}

func newTxMonitor(lock *sync.Mutex, alg TxAlgorithm, transport Transport) *TxMonitor {
	return &TxMonitor{
		lock:      lock,
		ready:     sync.NewCond(lock),
		alg:       alg,
		transport: transport,
	}
}

func (txm *TxMonitor) setRetxCallback(c func()) {
	txm.retxCallback = c
}

func (txm *TxMonitor) start() {
	go txm.run()
}

func (txm *TxMonitor) add(wm *WireMessage) {
	retxMs, deadline := txm.retxDeadline()
	txm.waitlist.Add(wm, retxMs, deadline)
	txm.ready.Broadcast()
}

func (txm *TxMonitor) remove(wm *WireMessage) {
	txm.waitlist.Remove(wm)
}

func (txm *TxMonitor) close() {
	txm.closed = true
	txm.ready.Broadcast()
}

func (txm *TxMonitor) retxDeadline() (int, time.Time) {
	retxMs := txm.alg.RetxMs()
	deadline := time.Now().Add(time.Duration(retxMs) * time.Millisecond)
	return retxMs, deadline
}

func (txm *TxMonitor) run() {
	logrus.Info("started")
	defer logrus.Warn("exited")

	for {
		var headline time.Time
		var timeout time.Duration

		txm.lock.Lock()
		{
			for txm.waitlist.Size() < 1 && !txm.closed {
				txm.ready.Wait()
			}

			if txm.closed {
				txm.lock.Unlock()
				return
			}

			_, headline = txm.waitlist.Peek()
			timeout = time.Until(headline)
		}
		txm.lock.Unlock()

		time.Sleep(timeout)

		txm.lock.Lock()
		{
			if txm.waitlist.Size() > 0 {
				i := 0
				x := txm.waitlist.Size()
				for ; i < x; i ++ {
					_, t := txm.waitlist.Peek()
					delta := t.Sub(headline).Milliseconds()
					if delta <= int64(txm.alg.RetxBatchMs()) {
						wm, _ := txm.waitlist.Next()
						if wm.hasFlag(RTT) {
							util.WriteUint16(wm.buf.Data[dataStart:], uint16(time.Now().UnixNano()/int64(time.Millisecond)))
						}

						if err := writeWireMessage(wm, txm.transport); err != nil {
							logrus.Errorf("retx (%v)", err)
						}
						if txm.retxCallback != nil {
							txm.retxCallback()
						}

						retxMs, deadline := txm.retxDeadline()
						txm.waitlist.Add(wm, retxMs, deadline)

					} else {
						break
					}
				}
			}
		}
		txm.lock.Unlock()
	}
}
