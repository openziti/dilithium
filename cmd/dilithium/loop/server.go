package loop

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	loopCmd.AddCommand(loopServerCmd)
}

var loopServerCmd = &cobra.Command{
	Use:   "server <listenAddress>",
	Short: "Start loop server",
	Args:  cobra.ExactArgs(1),
	Run:   loopServer,
}

func loopServer(_ *cobra.Command, args []string) {
	logrus.Infof("start sender: %t", startSender)
}
