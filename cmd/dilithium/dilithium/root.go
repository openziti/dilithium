package dilithium

import (
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
	RootCmd.PersistentFlags().StringVarP(&SelectedProtocol, "protocol", "p", "westworld2", "Select underlying protocol (tcp, tls, quic, conduit, westworld, westworld2)")
}

var RootCmd = &cobra.Command{
	Use:   strings.TrimSuffix(filepath.Base(os.Args[0]), filepath.Ext(os.Args[0])),
	Short: "Controlled UDP Explosions",
	PersistentPreRun: func(_ *cobra.Command, _ []string) {
		if verbose {
			logrus.SetLevel(logrus.DebugLevel)
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
var doCpuProfile bool
var cpuProfile interface{ Stop() }
var doMemoryProfile bool
var memoryProfile interface{ Stop() }
var doMutexProfile bool
var mutexProfile interface { Stop() }

