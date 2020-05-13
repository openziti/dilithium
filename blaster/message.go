package blaster

import (
	"encoding/gob"
	"net"
)

func init() {
	gob.Register(cmsg{})
	gob.Register(chello{})
}

type cmsgPair struct {
	h    cmsg
	p    interface{}
	peer *net.UDPAddr
}

type cmsg struct {
	Seq int32
	Mt  cmsgType
}

type cmsgType uint8

const (
	Sync cmsgType = iota
	Hello
	Ok
	Close
)

type chello struct {
	Nonce string
}
