package generate

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/generated/storage"
	clusterValidation "github.com/stackrox/rox/pkg/cluster"
	"github.com/stackrox/rox/roxctl/common/flags"
)

func k8s() *cobra.Command {
	c := &cobra.Command{
		Use:   "k8s",
		Short: "K8s specifies that you want to deploy into a Kubernetes cluster",
		Long:  `K8s specifies that you want to deploy into a Kubernetes cluster`,
		RunE: func(c *cobra.Command, _ []string) error {
			cluster.Type = storage.ClusterType_KUBERNETES_CLUSTER
			if err := clusterValidation.Validate(&cluster); err.ToError() != nil {
				return err.ToError()
			}
			return fullClusterCreation(flags.Timeout(c))
		},
	}
	return c
}
