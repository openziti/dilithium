package echo

import (
	"github.com/michaelquigley/dilithium/cmd/dilithium/dilithium"
	"github.com/spf13/cobra"
)

func init() {
	dilithium.RootCmd.AddCommand(echoCmd)
}

var echoCmd = &cobra.Command{
	Use:   "echo",
	Short: "Use a dilithium conduit for simple echo",
}
