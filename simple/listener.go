package simple

import "net"

func Listen(addr *net.UDPAddr) net.Listener{
	return &listener{addr: addr}
}

func (self *listener) Accept() (net.Conn, error) {
	return nil, nil
}

func (self *listener) Close() error {
	return nil
}

func (self *listener) Addr() net.Addr {
	return self.addr
}

type listener struct {
	addr *net.UDPAddr
}

