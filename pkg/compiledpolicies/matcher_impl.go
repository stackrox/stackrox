package compiledpolicies

import (
	"github.com/stackrox/rox/generated/api/v1"
	deploymentMatcher "github.com/stackrox/rox/pkg/compiledpolicies/deployment/matcher"
	deploymentPredicate "github.com/stackrox/rox/pkg/compiledpolicies/deployment/predicate"
	"github.com/stackrox/rox/pkg/compiledpolicies/utils"
)

// matcherAdapter is an adapter to the old interfaces.
// It just aggregates all the functionality needed to satisfy it.
type matcherAdapter struct {
	policy *v1.Policy

	matcher   deploymentMatcher.Matcher
	predicate deploymentPredicate.Predicate

	whitelists []*v1.Whitelist
}

// ShouldProcess checks if any part of the deployment should be matched.
func (p *matcherAdapter) ShouldProcess(deployment *v1.Deployment) bool {
	if p.predicate == nil {
		// This means the policy is enabled, applies to all scopes, and has no whitelist.
		// I.E. applies to EVERYTHING.
		return true
	}
	return p.predicate(deployment)
}

// Match matches the deployment and all of the deployment's containers and images.
func (p *matcherAdapter) Match(deployment *v1.Deployment) []*v1.Alert_Violation {
	if p.matcher == nil {
		// This means the policy had no fields (and therefore can never be violated).
		// Should not happen, but hey, who knows.
		return nil
	}
	return p.matcher(deployment)
}

// GetProto returns the original policy proto used to create the MatcherAdapter.
func (p *matcherAdapter) GetProto() *v1.Policy {
	return p.policy
}

// Excluded returns an explanation if a deployment is whitelisted in the policy.
// This is why we need adapters, basically to back trace which whitelist causes us to skip the deployement.
func (p *matcherAdapter) Excluded(deployment *v1.Deployment) (excluded *v1.DryRunResponse_Excluded) {
	for _, whitelist := range p.whitelists {
		if !utils.WhitelistIsExpired(whitelist) && deploymentPredicate.MatchesWhitelist(whitelist.GetDeployment(), deployment) {
			return &v1.DryRunResponse_Excluded{
				Deployment: deployment.GetName(),
				Whitelist:  whitelist,
			}
		}
	}
	return
}

// GetEnforcementAction returns the appropriate enforcement action for deployment.
func (p *matcherAdapter) GetEnforcementAction(deployment *v1.Deployment, action v1.ResourceAction) (v1.EnforcementAction, string) {
	return newEnforcement(p.policy.GetEnforcement()).enforcementAction(deployment, action)
}
