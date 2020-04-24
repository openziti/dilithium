package echo

import (
	"bufio"
	"github.com/michaelquigley/dilithium/conduit"
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
	listenAddress, err := net.ResolveUDPAddr("udp", args[0])
	if err != nil {
		logrus.Fatalf("error resolving listen address [%s] (%v)", args[0], err)
	}

	listener, err := conduit.Listen(listenAddress)
	if err != nil {
		logrus.Fatalf("error listening (%v)", err)
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			logrus.Errorf("error accepting (%v)", err)
			continue
		}
		go echoServerHandler(conn)
	}
}

func echoServerHandler(conn net.Conn) {
	defer func() { _ = conn.Close() }()

	reader := bufio.NewReader(conn)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			logrus.Errorf("error reading (%v)")
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
