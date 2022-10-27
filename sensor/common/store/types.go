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

// NetworkPolicyStore provides functionality to find matching Network Policies given a deployment
// object.
//go:generate mockgen-wrapper
type NetworkPolicyStore interface {
	Size() int
	All() map[string]*storage.NetworkPolicy
	Get(id string) *storage.NetworkPolicy
	Upsert(ns *storage.NetworkPolicy)
	Find(namespace string, labels map[string]string) map[string]*storage.NetworkPolicy
	Delete(ID, ns string)
}

// ServiceAccountStore provides functionality to find image pull secrets by service account
//go:generate mockgen-wrapper
type ServiceAccountStore interface {
	Add(sa *storage.ServiceAccount)
	Remove(sa *storage.ServiceAccount)
	GetImagePullSecrets(namespace, name string) []string
}
