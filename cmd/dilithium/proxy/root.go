package proxy

import (
	"github.com/michaelquigley/dilithium/cmd/dilithium/dilithium"
	"github.com/spf13/cobra"
)

func init() {
	dilithium.RootCmd.AddCommand(proxyCmd)
}

var proxyCmd = &cobra.Command{
	Use:   "proxy",
	Short: "Use a dilithium conduit as a proxy",
}
