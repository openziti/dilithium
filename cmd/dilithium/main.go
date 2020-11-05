package main

import (
	"github.com/openziti/dilithium/cmd/dilithium/dilithium"
	_ "github.com/openziti/dilithium/cmd/dilithium/echo"
	_ "github.com/openziti/dilithium/cmd/dilithium/influx"
	_ "github.com/openziti/dilithium/cmd/dilithium/loop"
	_ "github.com/openziti/dilithium/cmd/dilithium/tunnel"
	"github.com/michaelquigley/pfxlog"
	"github.com/sirupsen/logrus"
)

func init() {
	pfxlog.Global(logrus.InfoLevel)
	pfxlog.SetPrefix("github.com/openziti/")
}

func main() {
	defer logrus.Infof("finished")

	if err := dilithium.RootCmd.Execute(); err != nil {
		logrus.Fatalf("error (%v)", err)
	}
}
