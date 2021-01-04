package metrics

import (
	"github.com/openziti/dilithium/cmd/dilithium/dilithium"
	"github.com/spf13/cobra"
)

func init() {
	dilithium.RootCmd.AddCommand(metricsCmd)
}

var metricsCmd = &cobra.Command{
	Use:   "metrics",
	Short: "Control metrics instances",
}
