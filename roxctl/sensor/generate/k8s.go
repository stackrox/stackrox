package generate

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/generated/storage"
	clusterValidation "github.com/stackrox/rox/pkg/cluster"
	"github.com/stackrox/rox/pkg/features"
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

	c.PersistentFlags().BoolVar(&cluster.AdmissionController, "admission-controller", false, "whether or not to use an admission controller for enforcement")
	if features.AdmissionControlEnforceOnUpdate.Enabled() {
		c.PersistentFlags().BoolVar(&cluster.AdmissionControllerUpdates, "admission-controller-listen-on-updates", false, "whether or not to configure the admission controller webhook to listen on object updates")
	}

	// Admission controller config
	ac := cluster.DynamicConfig.AdmissionControllerConfig
	c.PersistentFlags().BoolVar(&ac.Enabled, "admission-controller-enabled", false, "dynamic enable for the admission controller")
	c.PersistentFlags().Int32Var(&ac.TimeoutSeconds, "admission-controller-timeout", 3, "timeout in seconds for the admission controller")
	c.PersistentFlags().BoolVar(&ac.ScanInline, "admission-controller-scan-inline", false, "get scans inline when using the admission controller")
	c.PersistentFlags().BoolVar(&ac.DisableBypass, "admission-controller-disable-bypass", false, "disable the bypass annotations for the admission controller")
	if features.AdmissionControlEnforceOnUpdate.Enabled() {
		c.PersistentFlags().BoolVar(&ac.EnforceOnUpdates, "admission-controller-enforce-on-updates", false, "dynamic enable for enforcing on object updates in the admission controller")
	}

	return c
}
