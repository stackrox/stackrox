package fake

import "time"

type deploymentWorkload struct {
	DeploymentType string
	NumDeployments int

	PodWorkload podWorkload

	UpdateInterval    time.Duration
	LifecycleDuration time.Duration
}

type processWorkload struct {
	ProcessInterval time.Duration
	AlertRate       float32
}

type networkWorkload struct {
	FlowInterval time.Duration
	BatchSize    int
}

type podWorkload struct {
	NumPods           int
	NumContainers     int
	LifecycleDuration time.Duration

	ProcessWorkload processWorkload
}

type nodeWorkload struct {
	NumNodes int
}

type workload struct {
	DeploymentWorkload []deploymentWorkload
	NodeWorkload       nodeWorkload
	NetworkWorkload    networkWorkload
}
