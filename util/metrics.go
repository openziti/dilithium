package util

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

type MetricsId struct {
	Id     string            `json:"id"`
	Values map[string]string `json:"values"`
}

func WriteMetricsId(outPath string, values map[string]string) error {
	mid := &MetricsId{Id: "westworld2.1", Values: values}
	data, err := json.MarshalIndent(mid, "", "  ")
	if err != nil {
		return err
	}
	oF, err := os.OpenFile(filepath.Join(outPath, "metrics.id"), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return err
	}
	defer func() { _ = oF.Close() }()
	if _, err := oF.Write(data); err != nil {
		return err
	}
	return nil
}

func ReadMetricsId(path string) (*MetricsId, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	metricsId := &MetricsId{}
	if err = json.Unmarshal(data, metricsId); err != nil {
		return nil, err
	}
	return metricsId, nil
}

type Sample struct {
	Ts time.Time
	V  int64
}

func WriteSamples(name, outPath string, samples []*Sample) error {
	path := filepath.Join(outPath, fmt.Sprintf("%s.csv", name))
	oF, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return err
	}
	defer func() { _ = oF.Close() }()
	for _, sample := range samples {
		line := fmt.Sprintf("%d,%d\n", sample.Ts.UnixNano(), sample.V)
		n, err := oF.Write([]byte(line))
		if err != nil {
			return err
		}
		if n != len(line) {
			return errors.New("short write")
		}
	}
	logrus.Infof("wrote [%d] samples to [%s]", len(samples), path)
	return nil
}
