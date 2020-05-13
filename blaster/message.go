package blaster

type cmsgPair struct {
	h cmsg
	p interface{}
}

type cmsg struct {
	seq int32
	mt  cmsgType
}

type cmsgType uint8

const (
	Sync cmsgType = iota
	Hello
	Close
)

type chello struct {
	nonce string
}
