package dilithium

import (
	"github.com/michaelquigley/dilithium/protocol/westworld2"
	"github.com/pkg/profile"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
	"strings"
)

func init() {
	RootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")
	RootCmd.PersistentFlags().BoolVar(&doCpuProfile, "cpu", false, "Enable CPU profiling")
	RootCmd.PersistentFlags().BoolVar(&doMemoryProfile, "memory", false, "Enable memory profiling")
	RootCmd.PersistentFlags().BoolVar(&doMutexProfile, "mutex", false, "Enable mutex profiling")
	RootCmd.PersistentFlags().StringVarP(&SelectedProtocol, "protocol", "p", "westworld2", "Select underlying protocol (tcp, tls, quic, westworld2)")
	RootCmd.PersistentFlags().StringVarP(&SelectedWestworld2Instrument, "instrument", "i", "logger", "Select westworld2 instrument (logger, nil)")
}

var RootCmd = &cobra.Command{
	Use:   strings.TrimSuffix(filepath.Base(os.Args[0]), filepath.Ext(os.Args[0])),
	Short: "Controlled UDP Explosions",
	PersistentPreRun: func(_ *cobra.Command, _ []string) {
		if verbose {
			logrus.SetLevel(logrus.DebugLevel)
		}
		switch SelectedWestworld2Instrument {
		case "logger":
			Westworld2Instrument = &westworld2.LoggerInstrument{}
		case "nil":
			Westworld2Instrument = nil
		default:
			logrus.Fatalf("unknown westworld2 logger instrument [%s]", SelectedWestworld2Instrument)
		}
		if doCpuProfile {
			cpuProfile = profile.Start(profile.CPUProfile)
		}
		if doMemoryProfile {
			memoryProfile = profile.Start(profile.MemProfile)
		}
		if doMutexProfile {
			mutexProfile = profile.Start(profile.MutexProfile)
		}
	},
	PersistentPostRun: func(_ *cobra.Command, _ []string) {
		if cpuProfile != nil {
			cpuProfile.Stop()
		}
		if memoryProfile != nil {
			memoryProfile.Stop()
		}
		if mutexProfile != nil {
			mutexProfile.Stop()
		}
	},
}
var verbose bool
var SelectedProtocol string
var SelectedWestworld2Instrument string
var doCpuProfile bool
var cpuProfile interface{ Stop() }
var doMemoryProfile bool
var memoryProfile interface{ Stop() }
var doMutexProfile bool
var mutexProfile interface{ Stop() }
