package metrics

import (
	"bufio"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"net"
	"strings"
)

func init() {
	metricsCmd.AddCommand(clientCmd)
}

var clientCmd = &cobra.Command{
	Use:   "client <path>",
	Short: "Connect to a metrics instance controller",
	Args:  cobra.ExactArgs(1),
	Run:   client,
}

func client(_ *cobra.Command, args []string) {
	path := args[0]
	addr, err := net.ResolveUnixAddr("unix", path)
	if err != nil {
		panic(err)
	}
	conn, err := net.DialUnix("unix", nil, addr)
	if err != nil {
		panic(err)
	}
	_, err = conn.Write([]byte("hello\n"))
	if err != nil {
		panic(err)
	}
	r := bufio.NewReader(conn)
	line, err := r.ReadString('\n')
	if err != nil {
		panic(err)
	}
	line = strings.TrimSpace(line)
	if line == "ok" {
		logrus.Infof("received 'ok'")
	} else {
		logrus.Errorf("invalid response '%s'", line)
	}
	_ = conn.Close()
}