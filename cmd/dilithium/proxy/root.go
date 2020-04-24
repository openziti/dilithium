package proxy

import "github.com/spf13/cobra"

var proxyCmd = &cobra.Command{
	Use:   "proxy",
	Short: "Use a dilithium conduit as a proxy",
}
