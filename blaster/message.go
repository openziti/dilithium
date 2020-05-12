package blaster

type cmsg struct {
	seq int32
	mt  cmsgType
}

type cmsgType uint8

const (
	Hello cmsgType = iota
	Close
)
