package main

import (
	"github.com/michaelquigley/pfxlog"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
	"strings"
)

func init() {
	pfxlog.Global(logrus.InfoLevel)
	pfxlog.SetPrefix("github.com/michaelquigley/")
	RootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")
}

var RootCmd = &cobra.Command{
	Use:   strings.TrimSuffix(filepath.Base(os.Args[0]), filepath.Ext(os.Args[0])),
	Short: "Controlled UDP Explosions",
	PersistentPreRun: func(_ *cobra.Command, _ []string) {
		if verbose {
			logrus.SetLevel(logrus.DebugLevel)
		}
	},
}
var verbose bool

func main() {
	if err := RootCmd.Execute(); err != nil {
		logrus.Fatalf("error (%v)", err)
	}
}
