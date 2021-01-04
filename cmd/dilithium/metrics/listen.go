package metrics

import (
	"github.com/openziti/dilithium/protocol/westworld3"
	"github.com/spf13/cobra"
	"time"
)

func init() {
	metricsCmd.AddCommand(listenCmd)
}

var listenCmd = &cobra.Command{
	Use:   "listen <root>",
	Short: "Listen for metrics clients",
	Args:  cobra.ExactArgs(1),
	Run:   listen,
}

func listen(_ *cobra.Command, args []string) {
	root := args[0]
	_, err := westworld3.NewMetricsInstrumentController(root, nil)
	if err != nil {
		panic(err)
	}
	for {
		time.Sleep(30 * time.Second)
	}
}
