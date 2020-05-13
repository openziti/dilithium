package blaster

import "encoding/gob"

func init() {
	gob.Register(cmsg{})
	gob.Register(chello{})
}

type cmsgPair struct {
	h cmsg
	p interface{}
}

type cmsg struct {
	Seq int32
	Mt  cmsgType
}

type cmsgType uint8

const (
	Sync cmsgType = iota
	Hello
	Close
)

type chello struct {
	Nonce string
}
