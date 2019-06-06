package generate

import (
	"encoding/json"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/apiparams"
	"github.com/stackrox/rox/pkg/roximages/defaults"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/common/zipdownload"
)

// Command represents the generate command.
func Command() *cobra.Command {
	var params apiparams.Scanner
	clusterType := storage.ClusterType_KUBERNETES_CLUSTER
	c := &cobra.Command{
		Use:   "generate",
		Short: "Generate creates the required YAML files to deploy StackRox Scanner.",
		Long:  "Generate creates the required YAML files to deploy StackRox Scanner.",
		RunE: func(c *cobra.Command, _ []string) error {
			params.ClusterType = clusterType.String()
			body, err := json.Marshal(params)
			if err != nil {
				return err
			}
			timeout := flags.Timeout(c)
			return zipdownload.GetZip("/api/extensions/scanner/zip", body, timeout, "scanner")
		},
	}

	c.Flags().StringVar(&params.ScannerImage, "scanner-image", defaults.ScannerImage(), "scanner image to use")
	c.Flags().BoolVar(&params.OfflineMode, "offline-mode", false, "whether to run the scanner in offline mode (so "+
		"it doesn't reach out to the internet for updates)")

	c.Flags().Var(clusterTypeWrapper{ClusterType: &clusterType}, "cluster-type", "type of cluster the scanner will run on (k8s, openshift)")
	return c
}
