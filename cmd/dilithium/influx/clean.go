package influx

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"io/ioutil"
	"net/http"
)

func init() {
	influxCmd.AddCommand(influxCleanCmd)
}

var influxCleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Clean metrics data from previous analyzer runs",
	Args:  cobra.NoArgs,
	Run:   influxClean,
}

func influxClean(_ *cobra.Command, args []string) {
	resp, err := http.Get(fmt.Sprintf("%s/query?db=%s&q=SHOW%%20SERIES", influxDbUrl, influxDbDatabase))
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	results := make(map[string]interface{})
	if err := json.Unmarshal(data, &results); err != nil {
		panic(err)
	}
	var series []string
	for _, v := range results["results"].([]interface{}) {
		rs := v.(map[string]interface{})
		if v, found := rs["series"]; found {
			for _, v := range v.([]interface{}) {
				rs := v.(map[string]interface{})
				if v, found := rs["values"]; found {
					for _, v := range v.([]interface{}) {
						for _, v := range v.([]interface{}) {
							series = append(series, v.(string))
						}
					}
				}
			}
		}
	}

	logrus.Infof("%v", series)
}
