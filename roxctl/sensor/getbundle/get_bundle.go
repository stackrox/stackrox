package getbundle

import (
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/sensor/util"
)

func downloadBundle(outputDir, clusterIDOrName string, createUpgraderSA bool, timeout time.Duration) error {
	clusterID, err := util.ResolveClusterID(clusterIDOrName, timeout)
	if err != nil {
		return err
	}

	if err := util.GetBundle(clusterID, outputDir, createUpgraderSA, timeout); err != nil {
		return errors.Wrap(err, "error getting cluster zip file")
	}
	return nil
}

// Command defines the deploy command tree
func Command() *cobra.Command {
	var createUpgraderSA bool
	var outputDir string

	c := &cobra.Command{
		Use: "get-bundle <cluster-name-or-id>",
		RunE: func(c *cobra.Command, args []string) error {
			if len(args) != 1 {
				_ = c.Help()
				return errors.Errorf("Expected exactly one argument, but %d were provided", len(args))
			}

			if err := downloadBundle(outputDir, args[0], createUpgraderSA, flags.Timeout(c)); err != nil {
				return errors.Wrap(err, "error downloading sensor bundle")
			}
			return nil
		},
	}

	c.PersistentFlags().StringVar(&outputDir, "output-dir", "", "output directory for bundle contents (default: auto-generated directory name inside the current directory)")
	c.PersistentFlags().BoolVar(&createUpgraderSA, "create-upgrader-sa", false, "whether to create the upgrader service account, with cluster-admin privileges, to facilitate automated sensor upgrades")

	return c
}
