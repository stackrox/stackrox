package generate

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/apiparams"
	"github.com/stackrox/rox/pkg/istioutils"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/common/util"
	"github.com/stackrox/rox/roxctl/scanner/clustertype"
	"github.com/stackrox/rox/roxctl/scanner/generate/run"
)

// Command represents the generate command.
func Command() *cobra.Command {
	var params apiparams.Scanner
	var opts run.Options

	c := &cobra.Command{
		Use: "generate",
		RunE: util.RunENoArgs(func(c *cobra.Command) error {
			return run.Run(c, &params, opts)
		}),
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
