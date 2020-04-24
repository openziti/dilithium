package tunnel

import (
	"github.com/michaelquigley/dilithium/conduit"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"net"
)

func init() {
	tunnelCmd.AddCommand(tunnelServerCmd)
}

var tunnelServerCmd = &cobra.Command{
	Use:   "server <listenAddress> <destinationTcpAddress>",
	Short: "Start tunnel server",
	Args:  cobra.ExactArgs(2),
	Run:   tunnelServer,
}

func tunnelServer(_ *cobra.Command, args []string) {
	listenAddress, err := net.ResolveUDPAddr("udp", args[0])
	if err != nil {
		logrus.Fatalf("error resolving listen address [%s] (%v)", args[0], err)
	}
	destinationAddress, err := net.ResolveTCPAddr("tcp", args[1])
	if err != nil {
		logrus.Fatalf("error resolving destination address [%s] (%v)", args[1], err)
	}

	listener, err := conduit.Listen(listenAddress)
	if err != nil {
		logrus.Fatalf("error creating listener (%v)", err)
	}
	logrus.Infof("listening at [%s]", listener.Addr())

	for {
		conn, err := listener.Accept()
		if err != nil {
			logrus.Errorf("error accepting (%v)", err)
			continue
		}
		go runTunnelTerminator(conn, destinationAddress)
	}
}

func runTunnelTerminator(iConn net.Conn, destinationAddress *net.TCPAddr) {
	defer func() { _ = iConn.Close() }()

	logrus.Infof("tunneling for [%s] to [%s]", iConn.RemoteAddr(), destinationAddress)
	defer logrus.Warnf("end tunnel for [%s]", iConn.RemoteAddr())

	tConn, err := net.DialTCP("tcp", nil, destinationAddress)
	if err != nil {
		logrus.Errorf("error connecting to destination [%s] (%v)", destinationAddress, err)
		return
	}
	go runTunnelTerminatorReader(iConn, tConn)
	defer func() { _ = tConn.Close() }()

	buffer := make([]byte, 10240)
	for {
		n, err := iConn.Read(buffer)
		if err != nil {
			logrus.Errorf("error reading from tunnel (%v)", err)
			return
		}
		n, err = tConn.Write(buffer[:n])
		if err != nil {
			logrus.Errorf("error writing to destination (%v)", err)
			return
		}
	}
}

func runTunnelTerminatorReader(iConn net.Conn, tConn net.Conn) {
	buffer := make([]byte, 10240)
	for {
		n, err := tConn.Read(buffer)
		if err != nil {
			logrus.Errorf("error reading from destination (%v)", err)
			return
		}
		n, err = iConn.Write(buffer[:n])
		if err != nil {
			logrus.Errorf("error writing to tunnel (%v)", err)
			return
		}
	}
}
