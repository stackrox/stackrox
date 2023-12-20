package cloudproviders

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stackrox/rox/pkg/logging"
)

const (
	aksClusterNameLabel = "kubernetes.azure.com/cluster"
	eksClusterNameLabel = "alpha.eksctl.io/cluster-name"
)

var log = logging.LoggerForModule()

// GetClusterMetadataFromNodeLabels returns the cluster metadata based on node labels.
func GetClusterMetadataFromNodeLabels(ctx context.Context) *storage.ClusterMetadata {
	k8sClient, err := k8sutil.GetK8sInClusterClient()
	if err != nil {
		log.Error("Failed to kubernetes client: ", err)
		return &storage.ClusterMetadata{}
	}
	nodeLabels, err := k8sutil.GetAnyNodeLabels(ctx, k8sClient)
	// TODO: remove this line
	log.Infof("All node labels: %+v", nodeLabels)
	if err != nil {
		log.Error("Failed to get node labels: ", err)
		return &storage.ClusterMetadata{}
	}

	if name, ok := nodeLabels[aksClusterNameLabel]; ok {
		return &storage.ClusterMetadata{Name: name, Type: storage.ClusterMetadata_AKS}
	}
	if name, ok := nodeLabels[eksClusterNameLabel]; ok {
		return &storage.ClusterMetadata{Name: name, Type: storage.ClusterMetadata_EKS}
	}
	return &storage.ClusterMetadata{}
}
