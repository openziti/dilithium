package influx

import (
	"bufio"
	"bytes"
	"fmt"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"
	"time"
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
	peers, err := discoverW21Peers(args[0])
	if err != nil {
		panic(err)
	}

	authToken := ""
	if influxDbUsername != "" || influxDbPassword != "" {
		authToken = fmt.Sprintf("%s:%s", influxDbUsername, influxDbPassword)
	}
	client := influxdb2.NewClient(influxDbUrl, authToken)
	writeApi := client.WriteAPI("", influxDbDatabase)

	for _, peer := range peers {
		for _, dataset := range datasets {
			data, err := readDataset(filepath.Join(peer.paths[0], dataset+".csv"))
			if err != nil {
				panic(err)
			}
			for ts, v := range data {
				t := time.Unix(0, ts)
				p := influxdb2.NewPoint(dataset, nil, map[string]interface{}{"v": v}, t).AddTag("peer", peer.id)
				writeApi.WritePoint(p)
			}
			logrus.Infof("wrote %d points for peer [%s] dataset [%s]", len(data), peer.id, dataset)
		}
	}

	client.Close()
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
