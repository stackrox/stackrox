package getbundle

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	pkgCommon "github.com/stackrox/rox/pkg/roxctl/common"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/sensor/util"
)

func resolveCluster(idOrName string, timeout time.Duration) (string, error) {
	if _, err := uuid.FromString(idOrName); err == nil {
		return idOrName, nil
	}

	conn, err := common.GetGRPCConnection()
	if err != nil {
		return "", err
	}

	service := v1.NewClustersServiceClient(conn)

	ctx, cancel := context.WithTimeout(pkgCommon.Context(), timeout)
	defer cancel()

	clusters, err := service.GetClusters(ctx, &v1.GetClustersRequest{
		Query: fmt.Sprintf("%s:%s", search.Cluster, idOrName),
	})
	if err != nil {
		return "", err
	}

	for _, cluster := range clusters.GetClusters() {
		if cluster.GetName() == idOrName {
			return cluster.GetId(), nil
		}
	}
	return "", errors.Errorf("no cluster with name %q found", idOrName)
}

func downloadBundle(outputDir, clusterIDOrName string, createUpgraderSA bool, timeout time.Duration) error {
	clusterID, err := resolveCluster(clusterIDOrName, timeout)
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
		Use:   "get-bundle <cluster-name-or-id>",
		Short: "Downloads the bundle with the required YAML files to deploy StackRox Sensor.",
		Long:  "Downloads the bundle with the required YAML files to deploy StackRox Sensor.",
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
