package ctrl

import (
	"bufio"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"net"
	"strings"
)

func init() {
	clientCmd.Flags().StringVarP(&clientCommand, "command", "c", "write", "Command to send")
	ctrlCmd.AddCommand(clientCmd)
}

var clientCmd = &cobra.Command{
	Use:   "client <path>",
	Short: "Connect to a metrics instance controller",
	Args:  cobra.ExactArgs(1),
	Run:   client,
}
var clientCommand string

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
	_, err = conn.Write([]byte(fmt.Sprintf("%s\n", clientCommand)))
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