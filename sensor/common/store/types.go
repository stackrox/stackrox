package store

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/sensor/common/rbac"
	"github.com/stackrox/rox/sensor/common/selector"
	"github.com/stackrox/rox/sensor/common/service"
)

// DeploymentStore provides functionality to fetch all deployments from underlying store.
//go:generate mockgen-wrapper
type DeploymentStore interface {
	GetAll() []*storage.Deployment
	Get(id string) *storage.Deployment
	FindDeploymentIDsWithServiceAccount(namespace, sa string) []string
	FindDeploymentIDsByLabels(namespace string, sel selector.Selector) []string
	BuildDeploymentWithDependencies(id string, dependencies Dependencies) (*storage.Deployment, error)
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

// ServiceStore provides functionality to find port exposure infos from in-memory services
//go:generate mockgen-wrapper
type ServiceStore interface {
	GetExposureInfos(namespace string, labels map[string]string) []map[service.PortRef][]*storage.PortConfig_ExposureInfo
}

// RBACStore provides functionality to find permission level from in-memory RBACs
//go:generate mockgen-wrapper
type RBACStore interface {
	GetPermissionLevelForDeployment(deployment rbac.NamespacedServiceAccount) storage.PermissionLevel
}

// Provider is a wrapper for injecting in memory stores as a dependency.
type Provider interface {
	Deployments() DeploymentStore
	Services() ServiceStore
	RBAC() RBACStore
	EndpointManager() EndpointManager
}

// EndpointManager provides functionality to map and store endpoints information
type EndpointManager interface {
	OnDeploymentCreateOrUpdateByID(id string)
}

// NodeStore provides functionality to get nodes
type NodeStore interface {
	AddOrUpdateNode(node *NodeWrap) bool
	GetNode(nodeName string) *NodeWrap
}
