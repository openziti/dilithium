package tunnel

import (
	"github.com/michaelquigley/dilithium/cmd/dilithium/dilithium"
	"github.com/spf13/cobra"
)

func init() {
	dilithium.RootCmd.AddCommand(tunnelCmd)
}

var tunnelCmd = &cobra.Command{
	Use:   "tunnel",
	Short: "Use a dilithium conduit as a tunnel",
}
