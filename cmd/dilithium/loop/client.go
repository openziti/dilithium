package loop

import "github.com/spf13/cobra"

func init() {
	loopCmd.AddCommand(loopClientCmd)
}

var loopClientCmd = &cobra.Command{
	Use:   "client <serverAddress>",
	Short: "Start loop client",
	Args:  cobra.ExactArgs(1),
	Run:   loopClient,
}

func loopClient(_ *cobra.Command, args []string) {
	//
}
