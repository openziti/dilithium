package influx

import (
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/openziti/dilithium/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"path/filepath"
	"time"
)

func loadWestworld31Metrics(root string, client influxdb2.Client) error {
	peer := westworld3PeerId(root)
	writeApi := client.WriteAPI("", influxDbDatabase)
	for _, dataset := range westworld31Datasets {
		datasetPath := filepath.Join(root, dataset+".csv")
		data, err := util.ReadSamples(datasetPath)
		if err != nil {
			return errors.Wrapf(err, "error reading dataset [%s]", datasetPath)
		}
		for ts, v := range data {
			t := time.Unix(0, ts)
			p := influxdb2.NewPoint(dataset, nil, map[string]interface{}{"v": v}, t).AddTag("type", "westworld31").AddTag("peer", peer)
			writeApi.WritePoint(p)
		}
		logrus.Infof("wrote [%d] points for westworld3.1 peer [%s] dataset [%s]", len(data), peer, dataset)
	}

	return nil
}

func westworld3PeerId(root string) string {
	return filepath.Base(root)
}

var westworld31Datasets = []string{
	"tx_bytes",
	"tx_msgs",
	"retx_bytes",
	"retx_msgs",
	"rx_bytes",
	"rx_msgs",
	"tx_portal_capacity",
	"tx_portal_sz",
	"tx_portal_rx_sz",
	"retx_ms",
	"dup_acks",
	"rx_portal_sz",
	"dup_rx_bytes",
	"dup_rx_msgs",
	"allocations",
	"errors",
}
