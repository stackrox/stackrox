package types

// SystemService is an abstraction for a container
type SystemService struct {
	Envs   []string
	Image  string
	Mounts []string
	Global bool
}

// Orchestrator is the interface that allows for actions against an orchestrator
type Orchestrator interface {
	Launch(service SystemService) (string, error)
	Kill(id string) error
}
