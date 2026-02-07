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
	// UseImageCopies when true uses ImageCopies instead of ImageNames
	// This creates paired deployments with _orig and _copy images for each random selection
	UseImageCopies bool `yaml:"useImageCopies"`
}

// ProcessWorkload defines the rate of process generation
type ProcessWorkload struct {
	ProcessInterval time.Duration `yaml:"processInterval"`
	AlertRate       float32       `yaml:"alertRate"`
	ActiveProcesses bool          `yaml:"activeProcesses"`
}

// NetworkWorkload defines the rate and size of network flows
type NetworkWorkload struct {
	FlowInterval              time.Duration `yaml:"flowInterval"`
	BatchSize                 int           `yaml:"batchSize"`
	GenerateUnclosedEndpoints bool          `yaml:"generateUnclosedEndpoints"`
	// OpenPortReuseProbability is the probability of reusing an existing open endpoint
	// by a different process without closing the endpoint.
	// In releases 4.8 and older, this was not configurable and was always set to 1.0.
	OpenPortReuseProbability float64 `yaml:"openPortReuseProbability"`
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

// VirtualMachineWorkload defines the workload for VirtualMachine and VirtualMachineInstance CRDs.
// This is the unified config for both VM/VMI informer events AND VM index reports.
// Index reports are only generated when ReportInterval > 0, and they follow the VM lifecycle
// (reports are sent only while the VM is "alive" in the informer simulation).
type VirtualMachineWorkload struct {
	// PoolSize is the number of VM/VMI templates to maintain in the pool.
	// This controls how many unique VMs exist at any given time.
	PoolSize int `yaml:"poolSize"`
	// UpdateInterval is how often to update VM/VMI metadata (annotations, labels)
	UpdateInterval time.Duration `yaml:"updateInterval"`
	// LifecycleDuration is how long each VM/VMI lifecycle lasts before recreation
	LifecycleDuration time.Duration `yaml:"lifecycleDuration"`
	// NumLifecycles is the number of times to recreate VMs/VMIs (0 = infinite)
	NumLifecycles int `yaml:"numLifecycles"`
	// InitialReportDelay delays the first index report for each VM by a user-provided duration.
	// A ±20% jitter is always applied to spread the initial burst; when unset, the first index
	// report is sent immediately (no delay, no jitter) once prerequisites are ready.
	InitialReportDelay time.Duration `yaml:"initialReportDelay"`

	// ReportInterval is how often each VM sends an index report (0 = no reports).
	// Index reports are only sent while the VM is alive in the informer simulation.
	ReportInterval time.Duration `yaml:"reportInterval"`
	// NumPackages is the number of fake packages to include in each index report
	NumPackages int `yaml:"numPackages"`
}

// Workload is the definition of a scale workload
type Workload struct {
	DeploymentWorkload     []DeploymentWorkload    `yaml:"deploymentWorkload"`
	NetworkPolicyWorkload  []NetworkPolicyWorkload `yaml:"networkPolicyWorkload"`
	NodeWorkload           NodeWorkload            `yaml:"nodeWorkload"`
	NetworkWorkload        NetworkWorkload         `yaml:"networkWorkload"`
	RBACWorkload           RBACWorkload            `yaml:"rbacWorkload"`
	ServiceWorkload        ServiceWorkload         `yaml:"serviceWorkload"`
	VirtualMachineWorkload VirtualMachineWorkload  `yaml:"virtualMachineWorkload"`
	NumNamespaces          int                     `yaml:"numNamespaces"`
	MatchLabels            bool                    `yaml:"matchLabels"`
}
