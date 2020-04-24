package tunnel

import (
	"github.com/michaelquigley/dilithium/conduit"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"net"
)

func init() {
	tunnelCmd.AddCommand(tunnelClientCmd)
}

var tunnelClientCmd = &cobra.Command{
	Use:   "client <serverAddress> <listenTcpAddress>",
	Short: "Start tunnel client",
	Args:  cobra.ExactArgs(2),
	Run:   tunnelClient,
}

func tunnelClient(_ *cobra.Command, args []string) {
	serverAddress, err := net.ResolveUDPAddr("udp", args[0])
	if err != nil {
		logrus.Fatalf("error resolving server address [%s] (%v)", args[0], err)
	}
	listenAddress, err := net.ResolveTCPAddr("tcp", args[1])
	if err != nil {
		logrus.Fatalf("error resolving listen address [%s] (%v)", args[1], err)
	}

	listener, err := net.ListenTCP("tcp", listenAddress)
	if err != nil {
		logrus.Infof("error creating listener at [%s] (%v)", listenAddress, err)
	}
	logrus.Infof("created listener at [%s]", listener.Addr())

	for {
		conn, err := listener.Accept()
		if err != nil {
			logrus.Errorf("error accepting (%v)", err)
			continue
		}
		go runTunnelClient(conn, serverAddress)
	}
}

func runTunnelClient(clientConn net.Conn, serverAddress *net.UDPAddr) {
	defer func() {
		_ = clientConn.Close()
	}()

	logrus.Infof("tunneling for [%s]", clientConn.RemoteAddr())
	defer logrus.Warnf("end tunnel for [%s]", clientConn.RemoteAddr())

	conduitConn, err := conduit.Dial(serverAddress)
	if err != nil {
		logrus.Errorf("error dialing server at [%s] (%v)", serverAddress, err)
	}
	defer func() {
		_ = conduitConn.Close()
	}()
	logrus.Infof("conduit established to [%s]", serverAddress)
}
