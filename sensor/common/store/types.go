package store

import "github.com/stackrox/rox/generated/storage"

// DeploymentStore provides functionality to fetch all deployments from underlying store.
//go:generate mockgen-wrapper
type DeploymentStore interface {
	GetAll() []*storage.Deployment
	Get(id string) *storage.Deployment
}

// PodStore provides functionality to fetch all pods from underlying store.
//go:generate mockgen-wrapper
type PodStore interface {
	GetAll() []*storage.Pod
	GetByName(podName, namespace string) *storage.Pod
}
