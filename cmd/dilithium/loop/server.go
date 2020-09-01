package loop

import (
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
	//
}
