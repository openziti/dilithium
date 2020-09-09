package loop

import (
	"github.com/michaelquigley/dilithium/util"
	"github.com/sirupsen/logrus"
	"time"
)

type transferReport struct {
	stamp time.Time
	bytes int64
}

type transferReporter struct {
	in         chan *transferReport
	pending    []*transferReport
	lastReport time.Time
}

func newTransferReporter() *transferReporter {
	return &transferReporter{
		in: make(chan *transferReport, 1024),
	}
}

func (self *transferReporter) run() {
	logrus.Info("started")
	defer logrus.Warn("exited")

	self.lastReport = time.Now()

	for {
		done := false
		for !done {
			select {
			case tr, ok := <-self.in:
				if ok {
					self.pending = append(self.pending, tr)
				} else {
					return
				}
			default:
				done = true
			}
		}

		now := time.Now()
		if now.Sub(self.lastReport).Milliseconds() >= 1000 {
			totalBytes := int64(0)
			lastI := -1
			done = false
			for i := 0; i < len(self.pending) && !done; i++ {
				if (self.pending[i].stamp.UnixNano() / int64(time.Millisecond)) < (self.lastReport.UnixNano()/int64(time.Millisecond))+1000 {
					totalBytes += self.pending[i].bytes
					lastI = i
				} else {
					done = true
				}
			}
			self.pending = self.pending[lastI+1:]
			self.lastReport = now

			logrus.Infof("%s/sec [:%d, %d pending]", util.BytesToSize(totalBytes), lastI, len(self.pending))
		}
	}
}
