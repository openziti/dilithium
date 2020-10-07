package influx

import (
	"encoding/json"
	"github.com/michaelquigley/dilithium/protocol/westworld2"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"path/filepath"
)

func discoverW21Peers(path string) ([]*w21peer, error) {
	var metricsIdPaths []string
	err := filepath.Walk(path, func(walkPath string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !fi.IsDir() && filepath.Base(walkPath) == "metrics.id" {
			metricsIdPaths = append(metricsIdPaths, filepath.Join(path, walkPath))
			logrus.Infof("appending: %s", walkPath)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	var metricsIds []*westworld2.MetricsId
	for _, path := range metricsIdPaths {
		metricsId, err := loadW21MetricsId(path)
		if err != nil {
			return nil, err
		}
		metricsIds = append(metricsIds, metricsId)
		logrus.Infof("loaded: %v", metricsId)
	}

	return nil, nil
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
