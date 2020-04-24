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

func runTunnelClient(iConn net.Conn, serverAddress *net.UDPAddr) {
	defer func() { _ = iConn.Close() }()

	logrus.Infof("tunneling for [%s]", iConn.RemoteAddr())
	defer logrus.Warnf("end tunnel for [%s]", iConn.RemoteAddr())

	tConn, err := conduit.Dial(serverAddress)
	if err != nil {
		logrus.Errorf("error dialing server at [%s] (%v)", serverAddress, err)
		return
	}
	go runTunnelClientReader(iConn, tConn)
	defer func() { _ = tConn.Close() }()
	logrus.Infof("conduit established to [%s]", serverAddress)

	buffer := make([]byte, 10240)
	for {
		n, err := iConn.Read(buffer)
		if err != nil {
			logrus.Errorf("error reading from initiator (%v)", err)
			return
		}
		logrus.Infof("<-(i) [%d]", n)
		n, err = tConn.Write(buffer[:n])
		if err != nil {
			logrus.Errorf("error writing to tunnel (%v)", err)
			return
		}
		logrus.Infof("->(t) [%d]", n)
	}
}

func runTunnelClientReader(iConn net.Conn, tConn net.Conn) {
	buffer := make([]byte, 10240)
	for {
		n, err := tConn.Read(buffer)
		if err != nil {
			logrus.Errorf("error reading from tunnel (%v)", err)
		}
		logrus.Infof("<-(t) [%d]", n)
		n, err = iConn.Write(buffer[:n])
		if err != nil {
			logrus.Errorf("error writing to initiator (%v)", err)
			return
		}
		logrus.Infof("->(i) [%d]", n)
	}
}