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
	dumpTimeout = 2 * time.Minute
)

// dumpCommand allows pulling logs, profiles, and metrics
func dumpCommand(cliEnvironment environment.Environment) *cobra.Command {
	var (
		withLogs  bool
		outputDir string
	)

	c := &cobra.Command{
		Use: "dump",
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
			}, cliEnvironment.Logger())
		}),
	}
	flags.AddTimeoutWithDefault(c, dumpTimeout)
	c.Flags().BoolVar(&withLogs, "logs", false, "Include logs in Central dump")
	c.PersistentFlags().StringVar(&outputDir, "output-dir", "", "output directory for bundle contents (default: auto-generated directory name inside the current directory)")

	return c
}
