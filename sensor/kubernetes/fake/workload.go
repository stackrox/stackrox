package fake

import (
	"time"
)

// DeploymentWorkload defines a workload of deployment objects
type DeploymentWorkload struct {
	DeploymentType string `yaml:"deploymentType"`
	NumDeployments int    `yaml:"numDeployments"`

	NumLabels    int  `yaml:"numLabels"`
	RandomLabels bool `yaml:"randomLabels"`

	PodWorkload PodWorkload `yaml:"podWorkload"`

	UpdateInterval    time.Duration `yaml:"updateInterval"`
	LifecycleDuration time.Duration `yaml:"lifecycleDuration"`
	NumLifecycles     int           `yaml:"numLifecycles"`
}

// NetworkPolicyWorkload defines a workload of networkPolicy objects
type NetworkPolicyWorkload struct {
	NumNetworkPolicies int `yaml:"numNetworkPolicies"`

	UpdateInterval    time.Duration `yaml:"updateInterval"`
	LifecycleDuration time.Duration `yaml:"lifecycleDuration"`
	NumLifecycles     int           `yaml:"numLifecycles"`
}

// ContainerWorkload defines the workloads for the container within the Pod
type ContainerWorkload struct {
	NumImages int `yaml:"numImages"`
}

// ProcessWorkload defines the rate of process generation
type ProcessWorkload struct {
	ProcessInterval time.Duration `yaml:"processInterval"`
	AlertRate       float32       `yaml:"alertRate"`
	ActiveProcesses bool          `yaml:"activeProcesses"`
}

// NetworkWorkload defines the rate and size of network flows
type NetworkWorkload struct {
	FlowInterval time.Duration `yaml:"flowInterval"`
	BatchSize    int           `yaml:"batchSize"`
}

// PodWorkload defines the workload and lifecycle of the pods within a deployment
type PodWorkload struct {
	NumPods           int           `yaml:"numPods"`
	NumContainers     int           `yaml:"numContainers"`
	LifecycleDuration time.Duration `yaml:"lifecycleDuration"`

	ContainerWorkload ContainerWorkload `yaml:"containerWorkload"`
	ProcessWorkload   ProcessWorkload   `yaml:"processWorkload"`
}

// NodeWorkload defines the node workload
type NodeWorkload struct {
	NumNodes int `yaml:"numNodes"`
}

// RBACWorkload defines the workload of roles, bindings, and service accounts
type RBACWorkload struct {
	NumRoles           int `yaml:"numRoles"`
	NumBindings        int `yaml:"numBindings"`
	NumServiceAccounts int `yaml:"numServiceAccounts"`
}

// ServiceWorkload defines the workload of services
type ServiceWorkload struct {
	NumClusterIPs    int `yaml:"numClusterIPs"`
	NumNodePorts     int `yaml:"numNodePorts"`
	NumLoadBalancers int `yaml:"numLoadBalancers"`
}

// Workload is the definition of a scale workload
type Workload struct {
	DeploymentWorkload    []DeploymentWorkload    `yaml:"deploymentWorkload"`
	NetworkPolicyWorkload []NetworkPolicyWorkload `yaml:"networkPolicyWorkload"`
	NodeWorkload          NodeWorkload            `yaml:"nodeWorkload"`
	NetworkWorkload       NetworkWorkload         `yaml:"networkWorkload"`
	RBACWorkload          RBACWorkload            `yaml:"rbacWorkload"`
	ServiceWorkload       ServiceWorkload         `yaml:"serviceWorkload"`
	NumNamespaces         int                     `yaml:"numNamespaces"`
	MatchLabels           bool                    `yaml:"matchLabels"`
}
