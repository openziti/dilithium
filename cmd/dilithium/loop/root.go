package loop

import (
	"github.com/michaelquigley/dilithium/cmd/dilithium/dilithium"
	"github.com/spf13/cobra"
)

func init() {
	loopCmd.PersistentFlags().BoolVarP(&startSender, "sender", "s", false, "Start a sender on connect")
	loopCmd.PersistentFlags().IntVarP(&size, "size", "z", 1024*1024, "Size of the data set (in bytes)")
	loopCmd.PersistentFlags().IntVarP(&count, "count", "c", 1024, "Send count for data set")
	dilithium.RootCmd.AddCommand(loopCmd)
}

var loopCmd = &cobra.Command{
	Use:   "loop",
	Short: "Loop back a socket for load and veracity measurements",
}
var startSender bool
var size int
var count int