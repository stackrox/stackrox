package orchestrator

// NodeScrapeConfig encapsulates the container runtime version and is master node return values
type NodeScrapeConfig struct {
	ContainerRuntimeVersion string
	IsMasterNode            bool
}

// Orchestrator returns an interface to interact with an orchestrator generically
//
//go:generate mockgen-wrapper
type Orchestrator interface {
	GetNodeScrapeConfig(nodeName string) (*NodeScrapeConfig, error)
}
