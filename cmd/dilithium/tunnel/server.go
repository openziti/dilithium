package tunnel

import (
	"github.com/michaelquigley/dilithium/conduit"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"net"
	"time"
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

func runTunnelTerminator(conn net.Conn, destinationAddress *net.TCPAddr) {
	defer func() { _ = conn.Close() }()

	logrus.Infof("tunneling for [%s]", conn.RemoteAddr())
	defer logrus.Warnf("end tunnel for [%s]", conn.RemoteAddr())

	for {
		time.Sleep(30 * time.Second)
	}
}
