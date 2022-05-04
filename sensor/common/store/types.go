package store

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// DeploymentStore provides functionality to fetch all deployments from underlying store.
//go:generate mockgen-wrapper
type DeploymentStore interface {
	GetAll() []*storage.Deployment
	Get(id string) *storage.Deployment
	AddOrUpdateDeployment(DeploymentWrap)
	RemoveDeployment(wrap DeploymentWrap)
	GetDeploymentsByIDs(string, set.StringSet) []DeploymentWrap
	GetMatchingDeployments(namespace string, sel Selector) []DeploymentWrap
	OnNamespaceDeleted(namespace string)
	CountDeploymentsForNamespace(namespace string) int
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

// DeploymentWrap provides interface to the deploymentWrap
type DeploymentWrap interface {
	GetNamespace() string
	GetId() string
	GetOriginal() interface{}
	GetType() string
	GetPodLabels() map[string]string
	GetDeployment() *storage.Deployment
	GetPortConfigs() map[PortRef]*storage.PortConfig
	GetPods() []*v1.Pod
	GetStateTimestamp() int64
	AnyNonHostPort() bool
	Clone() DeploymentWrap
	UpdatePortExposureFromServices(svcs ...ServiceWithRoutes)
	UpdatePortExposure(svc ServiceWithRoutes)
	ToEvent(action central.ResourceAction) *central.SensorEvent
}

// Selector is a restricted version of selectorWrap
type Selector interface {
	Matches(LabelsWithLen) bool
}

// LabelsWithLen is label.Labels with added Len() function
type LabelsWithLen interface {
	Has(label string) (exists bool)
	Get(label string) (value string)
	Len() uint
}

// PortRef interface to portRef
type PortRef interface {
	GetPort() intstr.IntOrString
	GetProtocol() v1.Protocol
}

// ServiceWithRoutes interface to serviceWithRoutes
type ServiceWithRoutes interface {
	Exposure() map[PortRef][]*storage.PortConfig_ExposureInfo
	GetSelector() Selector
	GetServiceWrap() ServiceWrap
	GetNamespace() string
}

// ServiceStore interface to serviceStore
type ServiceStore interface {
	GetMatchingServicesWithRoutes(namespace string, labels map[string]string) []ServiceWithRoutes
	OnNamespaceDeleted(namespace string)
}

// ServiceWrap interface to serviceWrap
type ServiceWrap interface {
	GetService() *v1.Service
	GetSpec() v1.ServiceSpec
	GetNamespace() string
	GetSelector() Selector
}
