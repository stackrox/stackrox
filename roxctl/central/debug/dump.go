package debug

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/common/util"
	"github.com/stackrox/rox/roxctl/common/zipdownload"
)

const (
	dumpTimeout = 2 * time.Minute
)

// DumpCommand allows pulling logs, profiles, and metrics
func DumpCommand() *cobra.Command {
	var (
		withLogs  bool
		outputDir string
	)

	c := &cobra.Command{
		Use:   "dump",
		Short: "Retrieve a zip file containing debug metrics",
		Long:  "Retrieve a zip file containing debug metrics",
		RunE: util.RunENoArgs(func(c *cobra.Command) error {
			fmt.Fprint(os.Stderr, "Retrieving debug metrics. This may take a couple minutes...\n")
			return retrieveDump(flags.Timeout(c), withLogs, outputDir)
		}),
	}
	flags.AddTimeoutWithDefault(c, dumpTimeout)
	c.Flags().BoolVar(&withLogs, "logs", false, "logs=true will retrieve the dump without logs from Central")
	c.PersistentFlags().StringVar(&outputDir, "output-dir", "", "output directory for bundle contents (default: auto-generated directory name inside the current directory)")

	return c
}

func retrieveDump(timeout time.Duration, logs bool, outputDir string) error {
	path := fmt.Sprintf("/debug/dump?logs=%t", logs)
	return zipdownload.GetZip(zipdownload.GetZipOptions{
		Path:       path,
		Method:     http.MethodGet,
		Timeout:    timeout,
		BundleType: "debug",
		ExpandZip:  false,
		OutputDir:  outputDir,
	})
}
