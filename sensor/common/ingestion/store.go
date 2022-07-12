package ingestion

import (
	"github.com/stackrox/rox/sensor/common/store"
	"github.com/stackrox/rox/sensor/kubernetes/listener/resources"
)

type ResourceStore struct {
	Deployments store.DeploymentStore
	NetworkPolicy store.NetworkPolicyStore
	PodStore store.PodStore
}

func NewStore() *ResourceStore {
	return &ResourceStore{
		Deployments: resources.NewDeploymentStore(),
		NetworkPolicy: resources.NewNetworkPolicyStore(),
		PodStore: resources.NewPodStore(),
	}
}
