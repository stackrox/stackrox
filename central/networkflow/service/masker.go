package service

import (
	"fmt"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/uuid"
)

const (
	// MaskedDeploymentName is name of a masked deployment
	MaskedDeploymentName = "masked deployment"
	// MaskedNamespaceName is name of a masked namespace
	MaskedNamespaceName = "masked namespace"
)

type flowGraphMasker struct {
	realToMaskedDeploymentID map[string]string
	realToMaskedNamespaceID  map[string]string
}

func newFlowGraphMasker() *flowGraphMasker {
	return &flowGraphMasker{
		realToMaskedDeploymentID: make(map[string]string),
		realToMaskedNamespaceID:  make(map[string]string),
	}
}

func (m *flowGraphMasker) GetMaskedDeploymentForCluster(clusterName string, namespace string, deploymentID string) *storage.Deployment {
	if _, ok := m.realToMaskedNamespaceID[namespace]; !ok {
		m.realToMaskedNamespaceID[namespace] = uuid.NewV4().String()
	}
	if _, ok := m.realToMaskedDeploymentID[deploymentID]; !ok {
		m.realToMaskedDeploymentID[deploymentID] = uuid.NewV4().String()
	}

	return &storage.Deployment{
		Id:          m.realToMaskedDeploymentID[deploymentID],
		Name:        MaskedDeploymentName,
		ClusterName: clusterName,
		Namespace:   fmt.Sprintf("%s:%s", MaskedNamespaceName, m.realToMaskedNamespaceID[namespace]),
		NamespaceId: m.realToMaskedNamespaceID[namespace],
	}
}

func (m *flowGraphMasker) GetFlowEntityForDeployment(deployment *storage.Deployment) *storage.NetworkEntityInfo {
	return &storage.NetworkEntityInfo{
		Id:   deployment.GetId(),
		Type: storage.NetworkEntityInfo_DEPLOYMENT,
		Desc: &storage.NetworkEntityInfo_Deployment_{
			Deployment: &storage.NetworkEntityInfo_Deployment{
				Name:      deployment.GetName(),
				Namespace: deployment.GetNamespace(),
				Cluster:   deployment.GetClusterName(),
			},
		},
	}
}
