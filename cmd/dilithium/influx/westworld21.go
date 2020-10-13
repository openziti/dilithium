package influx

import (
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/michaelquigley/dilithium/util"
	"github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func loadWestworld21Metrics(root string, client influxdb2.Client) error {
	peers, err := discoverW21Peers(root)
	if err != nil {
		panic(err)
	}

	writeApi := client.WriteAPI("", influxDbDatabase)
	for _, peer := range peers {
		for _, dataset := range westworld21Datasets {
			data, err := util.ReadSamples(filepath.Join(peer.paths[0], dataset+".csv"))
			if err != nil {
				panic(err)
			}
			for ts, v := range data {
				t := time.Unix(0, ts)
				p := influxdb2.NewPoint(dataset, nil, map[string]interface{}{"v": v}, t).AddTag("type", "westworld21").AddTag("peer", peer.id)
				writeApi.WritePoint(p)
			}
			logrus.Infof("wrote %d points for peer [%s] dataset [%s]", len(data), peer.id, dataset)
		}
	}

	return nil
}

func discoverW21Peers(path string) ([]*w21peer, error) {
	var metricsIdPaths []string
	err := filepath.Walk(path, func(walkPath string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !fi.IsDir() && filepath.Base(walkPath) == "metrics.id" {
			metricsIdPaths = append(metricsIdPaths, walkPath)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	var peers []*w21peer
	for _, path := range metricsIdPaths {
		metricsId, err := util.ReadMetricsId(path)
		if err != nil {
			return nil, err
		}
		if metricsId.Id == "westworld2.1" {
			parts := strings.Split(filepath.Base(filepath.Dir(path)), "_")
			peers = append(peers, &w21peer{
				id: parts[1],
				paths: []string{
					filepath.Dir(path),
				},
			})
		}
	}

	return peers, nil
}

type w21peer struct {
	id    string
	paths []string
}

var westworld21Datasets = []string{
	"txBytes",
	"txMsgs",
	"retxBytes",
	"retxMsgs",
	"rxBytes",
	"rxMsgs",
	"txPortalCapacity",
	"txPortalSz",
	"txPortalRxSz",
	"retxMs",
	"dupAcks",
	"rxPortalSz",
	"dupRxBytes",
	"dupRxMsgs",
	"allocations",
	"errors",
}
