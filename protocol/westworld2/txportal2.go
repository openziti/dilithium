package westworld2

import (
	"github.com/pkg/errors"
	"net"
)

type txPortal2 struct {
	txTree *txTree
	txRetx *txRetx
}

func newTxPortal2(conn *net.UDPConn, peer *net.UDPAddr, ins Instrument) *txPortal2 {
	tr := newTxRetx(conn, peer)
	return &txPortal2{
		txTree: newTxTree(tr.monitorIn, tr.cancelIn, conn, peer, ins),
		txRetx: tr,
	}
}

func (self *txPortal2) queueTx(wm *wireMessage) {
	self.txTree.txs <- wm
}

func (self *txPortal2) txError() error {
	select {
	case inerr, ok := <- self.txTree.errors:
		if !ok {
			return errors.New("closed")
		}
		return inerr
	default:
		// no queued error
	}
	return nil
}

func (self *txPortal2) ack(sequence int32) {
	self.txTree.acks <- sequence
}
