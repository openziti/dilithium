package influx

import (
	"github.com/openziti/dilithium/util"
	"github.com/pkg/errors"
	"path/filepath"
	"time"
)

func scanForLatestTimestamp(metricsMap map[string]*util.MetricsId) (int64, error) {
	latestTimestamp := time.Time{}
	for metricsRoot, metricsId := range metricsMap {
		switch metricsId.Id {
		case "westworld2.1":
			w21ts, err := findWestworld21LatestTimestamp(metricsRoot)
			if err != nil {
				return -1, err
			}
			if w21ts.After(latestTimestamp) {
				latestTimestamp = w21ts
			}

		case "westworld3.1":
			w31ts, err := findWestworld31LatestTimestamp(metricsRoot)
			if err != nil {
				return -1, err
			}
			if w31ts.After(latestTimestamp) {
				latestTimestamp = w31ts
			}

		case "dilithiumLoop":
			dlts, err := findDilithiumLoopLatestTimestamp(metricsRoot)
			if err != nil {
				return -1, err
			}
			if dlts.After(latestTimestamp) {
				latestTimestamp = dlts
			}
		}
	}
	if !latestTimestamp.Equal(time.Time{}) {
		retimeMs := time.Now().Sub(latestTimestamp).Milliseconds()
		return retimeMs, nil

	} else {
		return 0, nil
	}
}

func findLatestTimestamp(peers []*peer, datasets []string) (time.Time, error) {
	latestTimestamp := time.Time{}
	for _, peer := range peers {
		for _, dataset := range datasets {
			data, err := util.ReadSamples(filepath.Join(peer.paths[0], dataset+".csv"))
			if err != nil {
				return time.Time{}, errors.Wrapf(err, "error reading dataset [%s]", dataset)
			}
			for ts, _ := range data {
				t := time.Unix(0, ts)
				if t.After(latestTimestamp) {
					latestTimestamp = t
				}
			}
		}
	}

	return latestTimestamp, nil
}
