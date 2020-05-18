package westworld

import "github.com/michaelquigley/dilithium/protocol/westworld/pb"

type listenerConn struct{}

func (self *listenerConn) queue(wm *pb.WireMessage) {
}
