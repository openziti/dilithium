package util

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"time"
)

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