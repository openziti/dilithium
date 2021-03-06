package echo

import (
	"bufio"
	"github.com/openziti/dilithium/cmd/dilithium/dilithium"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"net"
)

func init() {
	echoCmd.AddCommand(echoServerCmd)
}

var echoServerCmd = &cobra.Command{
	Use:   "server <listenAddress>",
	Short: "Start echo server",
	Args:  cobra.ExactArgs(1),
	Run:   echoServer,
}

func echoServer(_ *cobra.Command, args []string) {
	protocol, err := dilithium.ProtocolFor(dilithium.SelectedProtocol)
	if err != nil {
		logrus.Fatalf("error selecting protocol (%v)", err)
	}

	listenAddress := args[0]
	listener, err := protocol.Listen(listenAddress)
	if err != nil {
		logrus.Fatalf("error listening (%v)", err)
	}
	logrus.Infof("listening at [%s]", listenAddress)

	for {
		conn, err := listener.Accept()
		if err != nil {
			logrus.Errorf("error accepting (%v)", err)
			continue
		}
		logrus.Infof("accepted connection from [%s]", conn.RemoteAddr())
		go echoServerHandler(conn)
	}
}

func echoServerHandler(conn net.Conn) {
	defer func() { _ = conn.Close() }()

	reader := bufio.NewReader(conn)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			logrus.Errorf("error reading (%v)", err)
			return
		}
		n, err := conn.Write(line)
		if err != nil {
			logrus.Errorf("error writing (%v)", err)
			return
		}
		if n != len(line) {
			logrus.Errorf("short write")
			return
		}
	}
}
