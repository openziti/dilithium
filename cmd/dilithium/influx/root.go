package influx

import (
	"github.com/michaelquigley/dilithium/cmd/dilithium/dilithium"
	"github.com/spf13/cobra"
)

func init() {
	influxCmd.PersistentFlags().StringVarP(&influxDbUrl, "url", "", "http://localhost:8086", "InfluxDB URL")
	influxCmd.PersistentFlags().StringVarP(&influxDbUsername, "username", "", "", "InfluxDB Username")
	influxCmd.PersistentFlags().StringVarP(&influxDbPassword, "password", "", "", "InfluxDB Password")
	influxCmd.PersistentFlags().StringVarP(&influxDbDatabase, "database", "", "dilithium", "InfluxDB Database")
	dilithium.RootCmd.AddCommand(influxCmd)
}

var influxCmd = &cobra.Command{
	Use: "influx",
	Short: "Manage the analyzer data in InfluxDB",
}
var influxDbUrl string
var influxDbUsername string
var influxDbPassword string
var influxDbDatabase string