package common

import (
	"time"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/common/flags"
)

const (
	defaultExportTimeout = 10 * time.Minute
)

// AddDefaultExportTimeout adds the timeout flag with the default export timeout
func AddDefaultExportTimeout(c *cobra.Command) {
	flags.AddTimeoutWithDefault(c, defaultExportTimeout)
}
