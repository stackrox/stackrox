package orchestrators

// Creator is a function stub that defined how to create a Orchestrator
type Creator func() (Orchestrator, error)

// SystemService is an abstraction for a container
type SystemService struct {
	Name    string
	Envs    []string
	Image   string
	Mounts  []string
	Global  bool
	Command []string
}

// Orchestrator is the interface that allows for actions against an orchestrator
type Orchestrator interface {
	Launch(service SystemService) (string, error)
	Kill(id string) error
}
