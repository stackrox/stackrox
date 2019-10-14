package run

import (
	"encoding/json"
	"net/http"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/apiparams"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/common/zipdownload"
	"github.com/stackrox/rox/roxctl/scanner/clustertype"
)

func getBundleType(params *apiparams.Scanner) string {
	if params.ScannerV2Config.Enable {
		return "scanner-v2"
	}
	return "scanner"
}

// Options stores options related to scanner generate commands.
type Options struct {
	OutputDir string
}

// Run extracts out the common logic from the scanner generate commands.
func Run(c *cobra.Command, params *apiparams.Scanner, opts Options) error {
	params.ClusterType = clustertype.Get().String()
	body, err := json.Marshal(params)
	if err != nil {
		return err
	}
	timeout := flags.Timeout(c)
	return zipdownload.GetZip(zipdownload.GetZipOptions{
		Path:       "/api/extensions/scanner/zip",
		Method:     http.MethodPost,
		Body:       body,
		Timeout:    timeout,
		BundleType: getBundleType(params),
		ExpandZip:  true,
		OutputDir:  opts.OutputDir,
	})
}
