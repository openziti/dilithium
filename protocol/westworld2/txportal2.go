package westworld2

import "net"

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
}

func (self *txPortal2) txError() error {
	return nil
}

func (self *txPortal2) ack(sequence int32) {
}
