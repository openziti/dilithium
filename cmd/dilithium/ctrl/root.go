package ctrl

import (
	"github.com/openziti/dilithium/cmd/dilithium/dilithium"
	"github.com/spf13/cobra"
)

func init() {
	dilithium.RootCmd.AddCommand(ctrlCmd)
}

var ctrlCmd = &cobra.Command{
	Use:   "ctrl",
	Short: "Control instances",
}
