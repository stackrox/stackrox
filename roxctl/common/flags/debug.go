package flags

import (
	"os"
	"path"

	"github.com/spf13/cobra"
)

var (
	debug          bool
	debugChartPath string
)

// AddDebug adds debug and debug-path flags to the base command.
func AddDebug(c *cobra.Command) {
	defaultDebugPath := path.Join(os.Getenv("GOPATH"), "src/github.com/stackrox/stackrox/image/")
	c.PersistentFlags().BoolVar(&debug, "debug", false, "read templates from local filesystem")
	c.PersistentFlags().StringVar(&debugChartPath, "debug-path", defaultDebugPath, "path to helm templates on your local filesystem")
}

// IsDebug returns whether debug flag is enabled
func IsDebug() bool {
	return debug
}

// DebugChartPath returns the path on the local filesystem to the chart to render
func DebugChartPath() string {
	return debugChartPath
}
