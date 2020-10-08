package influx

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
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
	serieses, err := influxShowSeries()
	if err != nil {
		panic(err)
	}
	for _, series := range serieses {
		if err := influxDropSeries(series); err == nil {
			logrus.Infof("dropped series [%s]", series)
		} else {
			panic(err)
		}
	}
}

func influxShowSeries() ([]string, error) {
	query := url.QueryEscape("SHOW SERIES")
	resp, err := http.Get(fmt.Sprintf("%s/query?db=%s&q=%s", influxDbUrl, influxDbDatabase, query))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	results := make(map[string]interface{})
	if err := json.Unmarshal(data, &results); err != nil {
		return nil, err
	}

	seriesMap := make(map[string]struct{})
	for _, v := range results["results"].([]interface{}) {
		rs := v.(map[string]interface{})
		if v, found := rs["series"]; found {
			for _, v := range v.([]interface{}) {
				rs := v.(map[string]interface{})
				if v, found := rs["values"]; found {
					for _, v := range v.([]interface{}) {
						for _, v := range v.([]interface{}) {
							series := v.(string)
							tokens := strings.Split(series, ",")
							seriesMap[tokens[0]] = struct{}{}
						}
					}
				}
			}
		}
	}
	var serieses []string
	for series, _ := range seriesMap {
		serieses = append(serieses, series)
	}
	return serieses, nil
}

func influxDropSeries(series string) error {
	query := url.QueryEscape(fmt.Sprintf("DROP SERIES FROM %s", series))
	resp, err := http.Post(fmt.Sprintf("%s/query?db=%s&q=%s", influxDbUrl, influxDbDatabase, query), "text/plain", nil)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return errors.Errorf("status(%d, %s)", resp.StatusCode, resp.Status)
	}
	return nil
}