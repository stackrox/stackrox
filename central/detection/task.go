package detection

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/compiledpolicies"
)

// Task describes a unit to be processed
type Task struct {
	deployment *v1.Deployment
	action     v1.ResourceAction
	policy     compiledpolicies.DeploymentMatcher
}

// NewTask creates a new task object
func NewTask(deployment *v1.Deployment, action v1.ResourceAction, policy compiledpolicies.DeploymentMatcher) Task {
	return Task{
		deployment: deployment,
		action:     action,
		policy:     policy,
	}
}
