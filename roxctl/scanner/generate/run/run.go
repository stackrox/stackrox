package run

import (
	"encoding/json"

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

// Run extracts out the common logic from the scanner generate commands.
func Run(c *cobra.Command, params *apiparams.Scanner) error {
	params.ClusterType = clustertype.Get().String()
	body, err := json.Marshal(params)
	if err != nil {
		return err
	}
	timeout := flags.Timeout(c)
	return zipdownload.GetZip("/api/extensions/scanner/zip", body, timeout, getBundleType(params))
}
