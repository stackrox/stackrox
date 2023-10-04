package debug

import (
	"fmt"
	"net/http"
	"time"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/common/util"
	"github.com/stackrox/rox/roxctl/common/zipdownload"
)

const (
	dumpTimeout = 5 * time.Minute
)

// dumpCommand allows pulling logs, profiles, and metrics
func dumpCommand(cliEnvironment environment.Environment) *cobra.Command {
	var (
		withLogs  bool
		outputDir string
	)

	c := &cobra.Command{
		Use:   "dump",
		Short: "Download a bundle containing debug information for Central.",
		Long:  "Download a bundle containing debug information for Central such as log files, memory, and CPU profiles. Bundle generation takes a few minutes.",
		RunE: util.RunENoArgs(func(c *cobra.Command) error {
			cliEnvironment.Logger().InfofLn("Retrieving debug metrics. This may take a couple of minutes...")
			path := fmt.Sprintf("/debug/dump?logs=%t", withLogs)
			return zipdownload.GetZip(zipdownload.GetZipOptions{
				Path:       path,
				Method:     http.MethodGet,
				Timeout:    flags.Timeout(c),
				BundleType: "debug",
				ExpandZip:  false,
				OutputDir:  outputDir,
			}, cliEnvironment)
		}),
	}
	flags.AddTimeoutWithDefault(c, dumpTimeout)
	flags.AddRetryTimeoutWithDefault(c, time.Duration(0))
	c.Flags().BoolVar(&withLogs, "logs", false, "Include logs in Central dump")
	c.PersistentFlags().StringVar(&outputDir, "output-dir", "", "output directory for bundle contents (default: auto-generated directory name inside the current directory)")

	return c
}
