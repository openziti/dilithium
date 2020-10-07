package main

import (
	"github.com/michaelquigley/dilithium/cmd/dilithium/dilithium"
	_ "github.com/michaelquigley/dilithium/cmd/dilithium/echo"
	_ "github.com/michaelquigley/dilithium/cmd/dilithium/influx"
	_ "github.com/michaelquigley/dilithium/cmd/dilithium/loop"
	_ "github.com/michaelquigley/dilithium/cmd/dilithium/tunnel"
	"github.com/michaelquigley/pfxlog"
	"github.com/sirupsen/logrus"
)

func init() {
	pfxlog.Global(logrus.InfoLevel)
	pfxlog.SetPrefix("github.com/michaelquigley/")
}

func main() {
	defer logrus.Infof("finished")

	if err := dilithium.RootCmd.Execute(); err != nil {
		logrus.Fatalf("error (%v)", err)
	}
}
