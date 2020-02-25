package data

import "time"

// NodeResourceInfo contains telemetry data about the resources belonging to a node in a Kubernetes cluster
type NodeResourceInfo struct {
	MilliCores   int   `json:"millicores"`
	MemoryBytes  int64 `json:"memoryBytes"`
	StorageBytes int64 `json:"storageBytes"`
}

// NodeInfo contains telemetry data about a node in a Kubernetes cluster
type NodeInfo struct {
	ID string `json:"id"`

	ProviderType         string            `json:"providerType,omitempty"`
	TotalResources       *NodeResourceInfo `json:"totalResources,omitempty"`
	AllocatableResources *NodeResourceInfo `json:"allocatableResources,omitempty"`
	Unschedulable        bool              `json:"unschedulable,omitempty"`
	HasTaints            bool              `json:"hasTaints,omitempty"`
	AdverseConditions    []string          `json:"adverseConditions,omitempty"`

	KernelVersion           string `json:"kernelVersion"`
	OSImage                 string `json:"osImage"`
	ContainerRuntimeVersion string `json:"containerRuntimeVersion"`
	KubeletVersion          string `json:"kubeletVersion"`
	KubeProxyVersion        string `json:"kubeProxyVersion"`
	OperatingSystem         string `json:"operatingSystem"`
	Architecture            string `json:"arch"`

	Collector  *RoxComponentInfo `json:"collector,omitempty"`
	Compliance *RoxComponentInfo `json:"compliance,omitempty"`
}

// NamespaceInfo contains telemetry data about a namespace in a Kubernetes cluster
type NamespaceInfo struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`

	NumPods        int `json:"numPods"`
	NumDeployments int `json:"numDeployments"`

	PodChurn        int `json:"podChurn"`
	DeploymentChurn int `json:"deploymentChurn"`
}

// OrchestratorInfo contains information about an orchestrator
type OrchestratorInfo struct {
	Orchestrator        string `json:"orchestrator"`
	OrchestratorVersion string `json:"orchestratorVersion"`
}

// SensorInfo contains information about a sensor and the cluster it is monitoring
type SensorInfo struct {
	*RoxComponentInfo

	LastCheckIn        *time.Time `json:"lastCheckIn,omitempty"`
	CurrentlyConnected bool       `json:"currentlyConnected"`
}

// ClusterInfo contains telemetry data about a Kubernetes cluster
type ClusterInfo struct {
	ID string `json:"id"`

	Sensor        *SensorInfo       `json:"sensor,omitempty"`
	Orchestrator  *OrchestratorInfo `json:"orchestrator,omitempty"`
	Nodes         []*NodeInfo       `json:"nodes,omitempty"`
	Namespaces    []*NamespaceInfo  `json:"namespaces,omitempty"`
	CloudProvider string            `json:"cloudProvider,omitempty"`
	Errors        []string          `json:"errors,omitempty"`
}
