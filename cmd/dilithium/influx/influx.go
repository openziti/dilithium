package influx

import (
	"bufio"
	"bytes"
	"github.com/michaelquigley/dilithium/cmd/dilithium/dilithium"
	"github.com/spf13/cobra"
	"io/ioutil"
	"strconv"
	"strings"
)

func init() {
	influxCmd.Flags().StringVarP(&influxDbUrl, "url", "", "http://localhost:8086", "InfluxDB URL")
	influxCmd.Flags().StringVarP(&influxDbUsername, "username", "", "", "InfluxDB Username")
	influxCmd.Flags().StringVarP(&influxDbPassword, "password", "", "", "InfluxDB Password")
	influxCmd.Flags().StringVarP(&influxDbDatabase, "database", "", "dilithium", "InfluxDB Database")
	dilithium.RootCmd.AddCommand(influxCmd)
}

var influxCmd = &cobra.Command{
	Use:   "influx <metricsRoot>",
	Short: "Import metrics data into the analyzer",
	Args:  cobra.ExactArgs(1),
	Run:   influx,
}
var influxDbUrl string
var influxDbUsername string
var influxDbPassword string
var influxDbDatabase string

func influx(_ *cobra.Command, args []string) {
	if _, err := discoverW21Peers(args[0]); err != nil {
		panic(err)
	}
	/*
	peer0Root := args[0]
	peer1Root := args[1]
	authToken := ""
	if influxDbUsername != "" || influxDbPassword != "" {
		authToken = fmt.Sprintf("%s:%s", influxDbUsername, influxDbPassword)
	}
	client := influxdb2.NewClient(influxDbUrl, authToken)
	writeApi := client.WriteApi("", influxDbDatabase)
	for _, dataset := range datasets {
		peer0Data, err := readDataset(filepath.Join(peer0Root, fmt.Sprintf("%s.csv", dataset)))
		if err != nil {
			logrus.Fatalf("error reading peer 0 dataset [%s] (%v)", dataset, err)
		}
		peer1Data, err := readDataset(filepath.Join(peer1Root, fmt.Sprintf("%s.csv", dataset)))
		if err != nil {
			logrus.Fatalf("error reading peer 1 dataset [%s] (%v)", dataset, err)
		}
		logrus.Infof("dataset [%s] loaded", dataset)

		for ts, v := range peer0Data {
			t := time.Unix(0, ts)
			p := influxdb2.NewPoint(dataset,
				nil,
				map[string]interface{}{"v": v},
				t,
			).AddTag("peer", "0")
			writeApi.WritePoint(p)
		}
		logrus.Infof("wrote [%d] points for peer 0 dataset [%s]", len(peer0Data), dataset)

		for ts, v := range peer1Data {
			t := time.Unix(0, ts)
			p := influxdb2.NewPoint(dataset,
				nil,
				map[string]interface{}{"v": v},
				t,
			).AddTag("peer", "1")
			writeApi.WritePoint(p)
		}
		logrus.Infof("wrote [%d] points for peer 1 dataset [%s]", len(peer1Data), dataset)
	}

	client.Close()
	logrus.Infof("complete")
	*/
}

func readDataset(path string) (data map[int64]int64, err error) {
	var raw []byte
	raw, err = ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	data = make(map[int64]int64)
	scanner := bufio.NewScanner(bytes.NewBuffer(raw))
	for scanner.Scan() {
		line := scanner.Text()
		tokens := strings.Split(line, ",")
		ts, err := strconv.ParseInt(tokens[0], 10, 64)
		if err != nil {
			return nil, err
		}
		v, err := strconv.ParseInt(tokens[1], 10, 64)
		if err != nil {
			return nil, err
		}
		data[ts] = v
	}

	return
}

var datasets = []string{
	"txBytes",
	"txMsgs",
	"retxBytes",
	"retxMsgs",
	"rxBytes",
	"rxMsgs",
	"txPortalCapacity",
	"txPortalSz",
	"retxMs",
	"dupAcks",
	"rxPortalSz",
	"dupRxBytes",
	"dupRxMsgs",
	"allocations",
	"errors",
}
