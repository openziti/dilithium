package tunnel

import (
	"github.com/openziti/dilithium/cmd/dilithium/dilithium"
	"github.com/openziti/dilithium/util"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"net"
	"runtime"
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
	cl, err := util.GetCtrlListener(".", "tunnel")
	if err != nil {
		logrus.Fatalf("error getting ctrl listener (%v)", err)
	}
	cl.AddCallback("stacks", func(_ string, conn net.Conn) (int64, error) {
		buf := make([]byte, 64*1024)
		n := runtime.Stack(buf, true)
		var err error
		n, err = conn.Write(buf[:n])
		return int64(n), err
	})
	cl.Start()

	protocol, err := dilithium.ProtocolFor(dilithium.SelectedProtocol)
	if err != nil {
		logrus.Fatalf("error selecting protocol (%v)", err)
	}

	listenAddress := args[0]
	destinationAddress, err := net.ResolveTCPAddr("tcp", args[1])
	if err != nil {
		logrus.Fatalf("error resolving destination address [%s] (%v)", args[1], err)
	}

	tunnelListener, err := protocol.Listen(listenAddress)
	if err != nil {
		logrus.Fatalf("error creating tunnel listener (%v)", err)
	}
	logrus.Infof("created tunnel listener at [%s]", listenAddress)

	for {
		conn, err := tunnelListener.Accept()
		if err != nil {
			logrus.Errorf("error accepting tunnel (%v)", err)
			continue
		}
		go handleTunnelTerminator(conn, destinationAddress)
	}
}

func handleTunnelTerminator(tunnel net.Conn, destinationAddress *net.TCPAddr) {
	defer func() { _ = tunnel.Close() }()

	logrus.Infof("tunneling for tunnel at [%s] to terminator at [%s]", tunnel.RemoteAddr(), destinationAddress)
	defer logrus.Warnf("end tunnel for [%s]", tunnel.RemoteAddr())

	terminator, err := net.DialTCP("tcp", nil, destinationAddress)
	if err != nil {
		logrus.Errorf("error connecting to terminator [%s] (%v)", destinationAddress, err)
		return
	}
	go handleTunnelTerminatorReader(tunnel, terminator)
	defer func() { _ = terminator.Close() }()

	buffer := make([]byte, bufferSize)
	for {
		n, err := tunnel.Read(buffer)
		if err != nil {
			logrus.Errorf("error reading from tunnel (%v)", err)
			return
		}
		n, err = terminator.Write(buffer[:n])
		if err != nil {
			logrus.Errorf("error writing to terminator (%v)", err)
			return
		}
	}
}

func handleTunnelTerminatorReader(tunnel net.Conn, terminator net.Conn) {
	buffer := make([]byte, bufferSize)
	for {
		n, err := terminator.Read(buffer)
		if err != nil {
			logrus.Errorf("error reading from terminator (%v)", err)
			return
		}
		n, err = tunnel.Write(buffer[:n])
		if err != nil {
			logrus.Errorf("error writing to tunnel (%v)", err)
			return
		}
	}
}
