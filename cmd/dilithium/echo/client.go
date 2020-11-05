package echo

import (
	"bufio"
	"fmt"
	"github.com/openziti/dilithium/cmd/dilithium/dilithium"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"net"
	"os"
)

func init() {
	echoCmd.AddCommand(echoClientCmd)
}

var echoClientCmd = &cobra.Command{
	Use:   "client <serverAddress>",
	Short: "Start echo client",
	Args:  cobra.ExactArgs(1),
	Run:   echoClient,
}

func echoClient(_ *cobra.Command, args []string) {
	protocol, err := dilithium.ProtocolFor(dilithium.SelectedProtocol)
	if err != nil {
		logrus.Fatalf("error selecting protocol (%v)", err)
	}

	serverAddress := args[0]
	conn, err := protocol.Dial(serverAddress)
	if err != nil {
		logrus.Fatalf("error dialing [%s] (%v)", serverAddress, err)
	}
	defer func() { _ = conn.Close() }()
	go echoClientReader(conn)
	logrus.Infof("connected to [%s]", serverAddress)

	input := bufio.NewReader(os.Stdin)
	for {
		line, err := input.ReadString('\n')
		if err != nil {
			logrus.Errorf("error reading console (%v)", err)
			break
		}
		n, err := conn.Write([]byte(line))
		if err != nil {
			logrus.Errorf("error writing network (%v)", err)
			break
		}
		if n != len([]byte(line)) {
			logrus.Errorf("short network write")
			break
		}
	}
}

func echoClientReader(conn net.Conn) {
	input := bufio.NewReader(conn)
	for {
		line, err := input.ReadString('\n')
		if err != nil {
			logrus.Errorf("error reading network (%v)", err)
			break
		}
		fmt.Printf(line)
	}
}
