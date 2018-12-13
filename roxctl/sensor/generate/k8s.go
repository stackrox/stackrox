package generate

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/generated/storage"
)

func k8s() *cobra.Command {
	c := &cobra.Command{
		Use:   "k8s",
		Short: "K8s specifies that you want to deploy into a Kubernetes cluster",
		Long:  `K8s specifies that you want to deploy into a Kubernetes cluster`,
		RunE: func(*cobra.Command, []string) error {
			cluster.Type = storage.ClusterType_KUBERNETES_CLUSTER
			return fullClusterCreation()
		},
	}
	commonParams := &storage.CommonKubernetesParams{}
	kubernetesParams := &storage.KubernetesParams{
		Params: commonParams,
	}
	cluster.OrchestratorParams = &storage.Cluster_Kubernetes{
		Kubernetes: kubernetesParams,
	}
	c.PersistentFlags().StringVarP(&commonParams.Namespace, "namespace", "n", "stackrox", "namespace to deploy into")
	return c
}
