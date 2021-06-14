package ctrl

import (
	"github.com/openziti/dilithium/util"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"net"
	"time"
)

func init() {
	ctrlCmd.AddCommand(listenCmd)
}

var listenCmd = &cobra.Command{
	Use:   "listen <root> <id>",
	Short: "Listen for clients",
	Args:  cobra.ExactArgs(2),
	Run:   listen,
}

func listen(_ *cobra.Command, args []string) {
	root := args[0]
	id := args[1]
	cl, err := util.GetCtrlListener(root, id)
	if err != nil {
		panic(err)
	}
	cl.AddCallback("hello", func(string, net.Conn) (int64, error) {
		logrus.Infof("invoked")
		return 0, nil
	})
	cl.AddCallback("hello", func(_ string, conn net.Conn) (int64, error) {
		logrus.Infof("oh, wow!")
		n, err := conn.Write([]byte("oh, wow!"))
		return int64(n), err
	})
	cl.Start()
	for {
		time.Sleep(30 * time.Second)
	}
}
