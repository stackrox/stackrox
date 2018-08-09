package compiledpolicies

import (
	"github.com/stackrox/rox/generated/api/v1"
	deploymentMatcher "github.com/stackrox/rox/pkg/compiledpolicies/deployment/matcher"
	deploymentPredicate "github.com/stackrox/rox/pkg/compiledpolicies/deployment/predicate"
)

// DeploymentMatcher is a bridge between current matching logic and the old matching interface.
// We simply build a top level predicate and matcher for the policy, and wrap it so that it matches the old behaviors.
type DeploymentMatcher interface {
	ShouldProcess(deployment *v1.Deployment) bool
	Match(deployment *v1.Deployment) []*v1.Alert_Violation

	GetProto() *v1.Policy
	Excluded(*v1.Deployment) *v1.DryRunResponse_Excluded
	GetEnforcementAction(*v1.Deployment, v1.ResourceAction) (v1.EnforcementAction, string)
}

// New returns a new DeploymentMatcher for the given policy.
func New(policy *v1.Policy) (DeploymentMatcher, error) {
	// Build the deployment matcher and its predicate to support matching and 'ShouldProcess'.
	matcher, err := deploymentMatcher.Compile(policy)
	if err != nil {
		return nil, err
	}
	predicate, err := deploymentPredicate.Compile(policy)
	if err != nil {
		return nil, err
	}

	// We need to keep the deployment whitelists to support the 'Excluded' function.
	whitelists := make([]*v1.Whitelist, 0)
	for _, whitelist := range policy.GetWhitelists() {
		if whitelist.GetDeployment() != nil {
			whitelists = append(whitelists, whitelist)
		}
	}

	// Build adapter.
	p := &matcherAdapter{
		policy:     policy,
		matcher:    matcher,
		predicate:  predicate,
		whitelists: whitelists,
	}
	return p, nil
}
