package enforcers

import (
	"github.com/stackrox/rox/generated/api/v1"
)

// Label key used for unsatisfiable node constraint enforcement.
const (
	UnsatisfiableNodeConstraintKey = `BlockedByStackRoxPrevent`
)

// DeploymentEnforcement wraps a request to take an enforcement action on a particular deployment.
type DeploymentEnforcement struct {
	Deployment   *v1.Deployment
	OriginalSpec interface{}
	Enforcement  v1.EnforcementAction
	AlertID      string
}

// EnforceFunc represents an enforcement function.
type EnforceFunc func(*DeploymentEnforcement) error

// Enforcer is an abstraction for taking enforcement actions on deployments.
type Enforcer interface {
	Actions() chan<- *DeploymentEnforcement
	Start()
	Stop()
}
