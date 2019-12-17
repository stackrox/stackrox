package generate

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/apiparams"
	"github.com/stackrox/rox/roxctl/scanner/clustertype"
	"github.com/stackrox/rox/roxctl/scanner/generate/run"
)

// Command represents the generate command.
func Command() *cobra.Command {
	var params apiparams.Scanner
	var opts run.Options

	c := &cobra.Command{
		Use:   "generate",
		Short: "Generate creates the required YAML files to deploy StackRox Scanner.",
		Long:  "Generate creates the required YAML files to deploy StackRox Scanner.",
		RunE: func(c *cobra.Command, _ []string) error {
			return run.Run(c, &params, opts)
		},
	}

	c.PersistentFlags().Var(clustertype.Value(storage.ClusterType_KUBERNETES_CLUSTER), "cluster-type", "type of cluster the scanner will run on (k8s, openshift)")

	c.Flags().StringVar(&opts.OutputDir, "output-dir", "", "Output directory for scanner bundle (leave blank for default)")
	c.Flags().BoolVar(&params.OfflineMode, "offline-mode", false, "whether to run the scanner in offline mode (so "+
		"it doesn't reach out to the internet for updates)")
	c.Flags().StringVar(&params.ScannerImage, "scanner-image", "", "Scanner image to use (leave blank to use server default)")

	return c
}
