package debug

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/common/util"
	"github.com/stackrox/rox/roxctl/common/zipdownload"
)

const (
	diagnosticBundleDownloadTimeout = 20 * time.Second
)

// downloadDiagnosticsCommand allows downloading the diagnostics bundle.
func downloadDiagnosticsCommand(cliEnvironment environment.Environment) *cobra.Command {
	var outputDir string
	var clusters []string
	var since string

	c := &cobra.Command{
		Use: "download-diagnostics",
		RunE: util.RunENoArgs(func(c *cobra.Command) error {
			cliEnvironment.Logger().InfofLn("Downloading diagnostic bundle...")
			path := "/api/extensions/diagnostics"

			values := url.Values{}
			for _, cluster := range clusters {
				values.Add("cluster", cluster)
			}
			if since != "" {
				values.Add("since", since)
			}

			urlParams := values.Encode()
			if urlParams != "" {
				path = fmt.Sprintf("%s?%s", path, urlParams)
			}
			return zipdownload.GetZip(cliEnvironment, zipdownload.GetZipOptions{
				Path:       path,
				Method:     http.MethodGet,
				Timeout:    flags.Timeout(c),
				BundleType: "diagnostic",
				ExpandZip:  false,
				OutputDir:  outputDir,
			})
		}),
	}
	flags.AddTimeoutWithDefault(c, diagnosticBundleDownloadTimeout)
	c.PersistentFlags().StringVar(&outputDir, "output-dir", "", "output directory in which to store bundle")
	c.PersistentFlags().StringSliceVar(&clusters, "clusters", nil, "comma separated list of sensor clusters from which logs should be collected")
	c.PersistentFlags().StringVar(&since, "since", "", "timestamp starting when logs should be collected from sensor clusters")

	return c
}
