package flags

import (
	"os"
	"path"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/image"
)

var (
	debug          bool
	debugChartPath string
)

// AddHelmChartDebugSetting adds debug and debug-path flags to the base command.
func AddHelmChartDebugSetting(c *cobra.Command) {
	defaultDebugPath := path.Join(os.Getenv("GOPATH"), "src/github.com/stackrox/stackrox/image/")
	c.PersistentFlags().BoolVar(&debug, "debug", false, "read templates from local filesystem")
	c.PersistentFlags().StringVar(&debugChartPath, "debug-path", defaultDebugPath, "path to helm templates on your local filesystem")
}

// IsDebug returns whether debug flag is enabled
func IsDebug() bool {
	return debug
}

// GetDebugHelmImage returns an image loaded from the local folder set by debugChartPath variable
func GetDebugHelmImage() *image.Image {
	return image.NewImage(os.DirFS(debugChartPath))
}
