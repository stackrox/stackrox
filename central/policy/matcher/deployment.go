package matcher

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/scopecomp"
	"github.com/stackrox/rox/pkg/utils"
)

type deploymentMatcher struct {
	deployment *storage.Deployment
}

// NewDeploymentMatcher creates a new policy matcher for deployment data.
func NewDeploymentMatcher(deployment *storage.Deployment) Matcher {
	return &deploymentMatcher{
		deployment: deployment,
	}
}

// FilterApplicablePolicies filters incoming policies into policies that apply to deployment and policies that do not apply to deployment
func (m *deploymentMatcher) FilterApplicablePolicies(policies []*storage.Policy) ([]*storage.Policy, []*storage.Policy) {
	applicable := make([]*storage.Policy, 0, len(policies)/2)
	notApplicable := make([]*storage.Policy, 0, len(policies)/2)

	for _, policy := range policies {
		if m.IsPolicyApplicable(policy) {
			applicable = append(applicable, policy)
		} else {
			notApplicable = append(notApplicable, policy)
		}
	}
	return applicable, notApplicable
}

// IsPolicyApplicable returns true if the policy is applicable to deployment
func (m *deploymentMatcher) IsPolicyApplicable(policy *storage.Policy) bool {
	return !policy.GetDisabled() && !m.anyWhitelistMatches(policy.GetWhitelists()) && m.anyScopeMatches(policy.GetScope())
}

func (m *deploymentMatcher) anyWhitelistMatches(whitelists []*storage.Whitelist) bool {
	for _, whitelist := range whitelists {
		if m.whitelistMatches(whitelist) {
			return true
		}
	}
	return false
}

func (m *deploymentMatcher) whitelistMatches(whitelist *storage.Whitelist) bool {
	// If whitelist scope does not match the deployment then no need to check for deployment name
	if !m.scopeMatches(whitelist.GetDeployment().GetScope()) {
		return false
	}

	// If scope of whitelist matches, the deployment is whitelisted if,
	// - Deployment name is not set, or
	// - Deployment name matches,
	return whitelist.GetDeployment().GetName() == "" ||
		whitelist.GetDeployment().GetName() == m.deployment.GetName()
}

func (m *deploymentMatcher) anyScopeMatches(scopes []*storage.Scope) bool {
	if len(scopes) == 0 {
		return true
	}

	for _, scope := range scopes {
		if m.scopeMatches(scope) {
			return true
		}
	}
	return false
}

func (m *deploymentMatcher) scopeMatches(scope *storage.Scope) bool {
	cs, err := scopecomp.CompileScope(scope)
	if err != nil {
		utils.Should(errors.Wrap(err, "could not compile scope"))
		return false
	}

	return cs.MatchesDeployment(m.deployment)
}
