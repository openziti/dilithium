package influx

import (
	"encoding/json"
	"github.com/michaelquigley/dilithium/protocol/westworld2"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

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
		metricsId, err := loadW21MetricsId(path)
		if err != nil {
			return nil, err
		}
		if metricsId.Id == "westworld2.1" {
			parts := strings.Split(filepath.Base(filepath.Dir(path)), "_")
			peers = append(peers, &w21peer{
				id: parts[1],
				paths: []string {
					filepath.Dir(path),
				},
			})
		}
	}

	return peers, nil
}

func loadW21MetricsId(path string) (*westworld2.MetricsId, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	metricsId := &westworld2.MetricsId{}
	if err = json.Unmarshal(data, metricsId); err != nil {
		return nil, err
	}
	return metricsId, nil
}

type w21peer struct {
	id    string
	paths []string
}
