package generate

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/generated/storage"
	clusterValidation "github.com/stackrox/rox/pkg/cluster"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/common/util"
)

func k8s() *cobra.Command {
	c := &cobra.Command{
		Use: "k8s",
		RunE: util.RunENoArgs(func(c *cobra.Command) error {
			cluster.Type = storage.ClusterType_KUBERNETES_CLUSTER
			cluster.DynamicConfig.DisableAuditLogs = true
			if err := clusterValidation.ValidatePartial(&cluster); err.ToError() != nil {
				return err.ToError()
			}
			return fullClusterCreation(flags.Timeout(c))
		}),
	}

	c.PersistentFlags().BoolVar(&cluster.AdmissionControllerEvents, "admission-controller-listen-on-events", true, "enable admission controller webhook to listen on Kubernetes events")
	return c
}
