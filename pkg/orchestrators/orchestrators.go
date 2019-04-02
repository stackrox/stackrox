package orchestrators

import (
	"time"

	"github.com/stackrox/rox/generated/storage"
)

// Creator is a function stub that defined how to create a Orchestrator
type Creator func() (Orchestrator, error)

// Secret is a generic definition of an orchestrator secret
type Secret struct {
	Name  string
	Items map[string]string

	TargetPath string
}

// SystemService is a generic definition of an orchestrator deployment
type SystemService struct {
	Name           string
	GenerateName   string
	ExtraPodLabels map[string]string
	Envs           []string
	SpecialEnvs    []SpecialEnvVar
	Image          string
	Mounts         []string
	Global         bool
	Resources      *storage.Resources
	Command        []string
	HostPID        bool
	ServiceAccount string
	Secrets        []Secret
	RunAsUser      *int64
}

// Orchestrator is the interface that allows for actions against an orchestrator
//go:generate mockgen-wrapper Orchestrator
type Orchestrator interface {
	Launch(service SystemService) (string, error)
	Kill(id string) error
	WaitForCompletion(service string, timeout time.Duration) error
	CleanUp(ownedByThisInstance bool) error
}
