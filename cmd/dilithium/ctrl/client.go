package ctrl

import (
	"bytes"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"io"
	"net"
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
	response := new(bytes.Buffer)
	_, err = io.Copy(response, conn)
	if err != nil {
		panic(err)
	}
	logrus.Infof("response:\n%s\n", string(response.Bytes()))
	_ = conn.Close()
}