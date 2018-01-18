package enforcers

import "bitbucket.org/stack-rox/apollo/generated/api/v1"

// DeploymentEnforcement wraps a request to take an enforcement action on a particular deployment.
type DeploymentEnforcement struct {
	Deployment   *v1.Deployment
	OriginalSpec interface{}
	Enforcement  v1.EnforcementAction
}

// EnforceFunc represents an enforcement function.
type EnforceFunc func(*DeploymentEnforcement) error

// Enforcer is an abstraction for taking enforcement actions on deployments.
type Enforcer interface {
	Actions() chan<- *DeploymentEnforcement
	Start()
	Stop()
}
