package util

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type MetricsId struct {
	Id     string            `json:"id"`
	Values map[string]string `json:"values,omitempty"`
}

func WriteMetricsId(id, outPath string, values map[string]string) error {
	mid := &MetricsId{Id: id, Values: values}
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

func DiscoverMetrics(root string) (map[string]*MetricsId, error) {
	var metricsIdPaths []string
	err := filepath.Walk(root, func(path string, fi os.FileInfo, err error) error  {
		if err != nil {
			return err
		}
		if !fi.IsDir() && filepath.Base(path) == "metrics.id" {
			metricsIdPaths = append(metricsIdPaths, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	metricsMap := make(map[string]*MetricsId)
	for _, metricsIdPath := range metricsIdPaths {
		metricsId, err := ReadMetricsId(metricsIdPath)
		if err != nil {
			return nil, errors.Wrapf(err, "error reading [%s]", metricsIdPath)
		}
		metricsRoot := filepath.Dir(metricsIdPath)
		metricsMap[metricsRoot] = metricsId
	}
	return metricsMap, nil
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

func ReadSamples(path string) (data map[int64]int64, err error) {
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