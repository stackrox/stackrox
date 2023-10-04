package debug

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/common/util"
	"github.com/stackrox/rox/roxctl/common/zipdownload"
)

const (
	diagnosticBundleDownloadTimeout = 300 * time.Second
)

// downloadDiagnosticsCommand allows downloading the diagnostics bundle.
func downloadDiagnosticsCommand(cliEnvironment environment.Environment) *cobra.Command {
	var outputDir string
	var clusters []string
	var since string

	c := &cobra.Command{
		Use:   "download-diagnostics",
		Short: "Download a bundle containing a snapshot of diagnostic information about the platform.",
		Long:  "Download a bundle containing a snapshot of diagnostic information such as logs from Central and Secured Clusters and other non-sensitive configuration data about the platform.",
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
			err := zipdownload.GetZip(zipdownload.GetZipOptions{
				Path:       path,
				Method:     http.MethodGet,
				Timeout:    flags.Timeout(c),
				BundleType: "diagnostic",
				ExpandZip:  false,
				OutputDir:  outputDir,
			}, cliEnvironment)
			if isTimeoutError(err) {
				cliEnvironment.Logger().ErrfLn(`Timeout has been reached while creating diagnostic bundle.
Timeout value used was %s, while default timeout value is %s.
If your timeout value is less than the default value, use the default value.
If your timeout value is more or equal to default value, increase timeout value twice in size.
To specify timeout, run  'roxctl' command:
'roxctl central debug download-diagnostics --timeout=<timeout> <other parameters'`, flags.Timeout(c), diagnosticBundleDownloadTimeout)
			}
			return err
		}),
	}
	flags.AddTimeoutWithDefault(c, diagnosticBundleDownloadTimeout)
	flags.AddRetryTimeoutWithDefault(c, time.Duration(0))
	c.PersistentFlags().StringVar(&outputDir, "output-dir", "", "output directory in which to store bundle")
	c.PersistentFlags().StringSliceVar(&clusters, "clusters", nil, "comma separated list of sensor clusters from which logs should be collected")
	c.PersistentFlags().StringVar(&since, "since", "", "timestamp starting when logs should be collected from sensor clusters")

	return c
}

func isTimeoutError(err error) bool {
	var timeoutErr httputil.TimeoutError

	return errors.As(err, &timeoutErr) && timeoutErr.Timeout() ||
		errors.Is(err, context.DeadlineExceeded)
}
