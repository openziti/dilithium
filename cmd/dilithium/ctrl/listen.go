package ctrl

import (
	"github.com/openziti/dilithium/util"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
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
	cl.AddCallback("hello", func(string) error {
		logrus.Infof("invoked")
		return nil
	})
	cl.AddCallback("hello", func(string) error {
		logrus.Infof("oh, wow!")
		return nil
	})
	cl.Start()
	for {
		time.Sleep(30 * time.Second)
	}
}
