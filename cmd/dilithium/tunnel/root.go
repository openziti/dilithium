package tunnel

import (
	"github.com/openziti/dilithium/cmd/dilithium/dilithium"
	"github.com/spf13/cobra"
)

const bufferSize = 16 * 1024

func init() {
	dilithium.RootCmd.AddCommand(tunnelCmd)
}

var tunnelCmd = &cobra.Command{
	Use:   "tunnel",
	Short: "Use a dilithium conduit as a tunnel",
}
