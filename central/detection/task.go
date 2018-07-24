package detection

import (
	"bitbucket.org/stack-rox/apollo/central/detection/matcher"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
)

// Task describes a unit to be processed
type Task struct {
	deployment *v1.Deployment
	action     v1.ResourceAction
	policy     *matcher.Policy
}

// NewTask creates a new task object
func NewTask(deployment *v1.Deployment, action v1.ResourceAction, policy *matcher.Policy) Task {
	return Task{
		deployment: deployment,
		action:     action,
		policy:     policy,
	}
}
