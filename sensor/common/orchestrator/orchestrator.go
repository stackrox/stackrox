package orchestrator

// Orchestrator returns an interface to interact with an orchestrator generically
type Orchestrator interface {
	GetNodeContainerRuntime(nodeName string) (string, error)
}
