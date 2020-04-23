package fireball

import "net"

type Conn struct {
	conn  *net.UDPConn
	local *net.UDPAddr
	peer  *net.UDPAddr
}

func (self *Conn) send(m *message) error {
	return nil
}