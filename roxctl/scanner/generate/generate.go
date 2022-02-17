package generate

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/apiparams"
	"github.com/stackrox/rox/pkg/istioutils"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/common/zipdownload"
	"github.com/stackrox/rox/roxctl/scanner/clustertype"
)

// Options stores options related to scanner generate commands.
type Options struct {
	OutputDir string
}

// Command represents the generate command.
func Command() *cobra.Command {
	var params apiparams.Scanner
	var opts Options

	c := &cobra.Command{
		Use:  "generate",
		Args: cobra.NoArgs,
		RunE: func(c *cobra.Command, args []string) error {
			params.ClusterType = clustertype.Get().String()
			body, err := json.Marshal(params)
			if err != nil {
				return errors.Wrap(err, "could not marshal scanner params")
			}
			timeout := flags.Timeout(c)
			return zipdownload.GetZip(zipdownload.GetZipOptions{
				Path:       "/api/extensions/scanner/zip",
				Method:     http.MethodPost,
				Body:       body,
				Timeout:    timeout,
				BundleType: "scanner",
				ExpandZip:  true,
				OutputDir:  opts.OutputDir,
			})
		},
	}

	c.PersistentFlags().Var(clustertype.Value(storage.ClusterType_KUBERNETES_CLUSTER), "cluster-type", "type of cluster the scanner will run on (k8s, openshift)")

	c.Flags().StringVar(&opts.OutputDir, "output-dir", "", "Output directory for scanner bundle (leave blank for default)")
	c.Flags().BoolVar(&params.OfflineMode, "offline-mode", false, "whether to run the scanner in offline mode (so "+
		"it doesn't reach out to the internet for updates)")
	c.Flags().StringVar(&params.ScannerImage, flags.FlagNameScannerImage, "", "Scanner image to use (leave blank to use server default)")
	c.Flags().StringVar(&params.IstioVersion, "istio-support", "",
		fmt.Sprintf(
			"Generate deployment files supporting the given Istio version. Valid versions: %s",
			strings.Join(istioutils.ListKnownIstioVersions(), ", ")))

	return c
}
