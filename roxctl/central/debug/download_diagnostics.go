package debug

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/common/zipdownload"
)

const (
	diagnosticBundleDownloadTimeout = 20 * time.Second
)

// DownloadDiagnosticsCommand allows downloading the diagnostics bundle.
func DownloadDiagnosticsCommand() *cobra.Command {
	var outputDir string

	c := &cobra.Command{
		Use:   "download-diagnostics",
		Short: `downloads a bundle with extended diagnostic information`,
		Long:  `downloads a bundle with extended diagnostic information`,
		RunE: func(c *cobra.Command, _ []string) error {
			fmt.Fprintln(os.Stderr, "Downloading diagnostic bundle...")
			return retrieveDiagnosticBundle(flags.Timeout(c), outputDir)
		},
	}
	flags.AddTimeoutWithDefault(c, diagnosticBundleDownloadTimeout)
	c.PersistentFlags().StringVar(&outputDir, "output-dir", "", "output directory in which to store bundle")

	return c
}

func retrieveDiagnosticBundle(timeout time.Duration, outputDir string) error {
	path := "/api/extensions/diagnostics"
	return zipdownload.GetZip(zipdownload.GetZipOptions{
		Path:       path,
		Method:     http.MethodGet,
		Timeout:    timeout,
		BundleType: "diagnostic",
		ExpandZip:  false,
		OutputDir:  outputDir,
	})
}
