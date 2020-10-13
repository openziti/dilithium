package influx

import (
	"fmt"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/spf13/cobra"
)

func init() {
	influxCmd.AddCommand(influxLoadCmd)
}

var influxLoadCmd = &cobra.Command{
	Use:   "load <metricsRoot>",
	Short: "Load metrics data into the analyzer",
	Args:  cobra.ExactArgs(1),
	Run:   influxLoad,
}

func influxLoad(_ *cobra.Command, args []string) {
	authToken := ""
	if influxDbUsername != "" || influxDbPassword != "" {
		authToken = fmt.Sprintf("%s:%s", influxDbUsername, influxDbPassword)
	}
	client := influxdb2.NewClient(influxDbUrl, authToken)

	if err := loadWestworld21Metrics(args[0], client); err != nil {
		panic(err)
	}
	if err := loadDilithiumLoopMetrics(args[0], client); err != nil {
		panic(err)
	}

	client.Close()
}
