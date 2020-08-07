package getbundle

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/pointers"
	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/pflag/autobool"
	"github.com/stackrox/rox/roxctl/sensor/util"
)

const (
	infoDefaultingToSlimCollector = `Specified cluster is configured to use a slim collector image, hence producing deployment bundle using a slim collector image.
Use --slim-collector=false if that is not desired.`
	infoDefaultingToComprehensiveCollector = `Specified cluster is configured to use a comprehensive collector image, hence producing deployment bundle using a comprehensive collector image.
Use --slim-collector if that is not desired.`
)

func downloadBundle(outputDir, clusterIDOrName string, createUpgraderSA bool, timeout time.Duration, slimCollectorP *bool) error {
	conn, err := common.GetGRPCConnection()
	if err != nil {
		return err
	}
	service := v1.NewClustersServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	clusterID, err := util.ResolveClusterID(clusterIDOrName, timeout)
	if err != nil {
		return errors.Wrapf(err, "error resolving cluster ID for %q", clusterIDOrName)
	}
	slimCollector := false
	if slimCollectorP != nil {
		slimCollector = *slimCollectorP
	} else {
		// Query Central for slimCollector property of the specified cluster.
		resp, err := service.GetCluster(ctx, &v1.ResourceByID{
			Id: clusterID,
		})
		if err != nil {
			return errors.Wrapf(err, "error resolving cluster for ID %q", clusterID)
		}
		cluster := resp.GetCluster()
		slimCollector = cluster.GetSlimCollector()
		if slimCollector {
			fmt.Fprintln(os.Stderr, infoDefaultingToSlimCollector)
		} else {
			fmt.Fprintln(os.Stderr, infoDefaultingToComprehensiveCollector)
		}
	}

	if slimCollector {
		env := util.RetrieveCentralEnvOrDefault(ctx, service)
		if !env.KernelSupportAvailable {
			fmt.Fprintf(os.Stderr, "%s\n\n", util.WarningSlimCollectorModeWithoutKernelSupport)
		}
	}

	if err := util.GetBundle(clusterID, outputDir, createUpgraderSA, timeout, slimCollector); err != nil {
		return errors.Wrap(err, "error getting cluster zip file")
	}
	return nil
}

// Command defines the deploy command tree
func Command() *cobra.Command {
	var createUpgraderSA bool
	var outputDir string
	var slimCollector *bool

	c := &cobra.Command{
		Use: "get-bundle <cluster-name-or-id>",
		RunE: func(c *cobra.Command, args []string) error {
			if len(args) != 1 {
				_ = c.Help()
				return errors.Errorf("Expected exactly one argument, but %d were provided", len(args))
			}

			if err := downloadBundle(outputDir, args[0], createUpgraderSA, flags.Timeout(c), slimCollector); err != nil {
				return errors.Wrap(err, "error downloading sensor bundle")
			}
			return nil
		},
	}

	c.PersistentFlags().StringVar(&outputDir, "output-dir", "", "output directory for bundle contents (default: auto-generated directory name inside the current directory)")
	c.PersistentFlags().BoolVar(&createUpgraderSA, "create-upgrader-sa", false, "whether to create the upgrader service account, with cluster-admin privileges, to facilitate automated sensor upgrades")

	if features.SupportSlimCollectorMode.Enabled() {
		autobool.NewFlag(c.PersistentFlags(), &slimCollector, "slim-collector", "Use slim collector in deployment bundle")
	} else {
		slimCollector = pointers.Bool(false)
	}

	return c
}
