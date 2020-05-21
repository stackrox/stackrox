package generate

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/generated/storage"
	clusterValidation "github.com/stackrox/rox/pkg/cluster"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/common/util"
)

func openshift() *cobra.Command {
	c := &cobra.Command{
		Use:   "openshift",
		Short: "openshift specifies that you want to deploy into an OpenShift cluster",
		Long:  `openshift specifies that you want to deploy into an OpenShift cluster`,
		RunE: util.RunENoArgs(func(c *cobra.Command) error {
			cluster.Type = storage.ClusterType_OPENSHIFT_CLUSTER
			if err := clusterValidation.Validate(&cluster).ToError(); err != nil {
				return err
			}
			return fullClusterCreation(flags.Timeout(c))
		}),
	}

	return c
}
