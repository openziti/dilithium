package westworld3

type wireMessage struct {
	seq int32
	mt  messageType
}

type messageType uint8

const (
	// 0x0 ... 0x7
	HELLO messageType = iota
	ACK
	DATA
	CLOSE
)

type messageFlag uint8

const (
	// 0x8 ... 0x80
	RTT messageFlag = 0x8
)
