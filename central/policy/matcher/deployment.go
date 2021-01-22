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
	return !policy.GetDisabled() && !m.anyExclusionMatches(policy.GetExclusions()) && m.anyScopeMatches(policy.GetScope())
}

func (m *deploymentMatcher) anyExclusionMatches(exclusions []*storage.Exclusion) bool {
	for _, exclusion := range exclusions {
		if m.exclusionMatches(exclusion) {
			return true
		}
	}
	return false
}

func (m *deploymentMatcher) exclusionMatches(exclusion *storage.Exclusion) bool {
	// If excluded scope does not match the deployment then no need to check for deployment name
	if !m.scopeMatches(exclusion.GetDeployment().GetScope()) {
		return false
	}

	// If scope of exclusions matches, the deployment is excluded if,
	// - Deployment name is not set, or
	// - Deployment name matches,
	return exclusion.GetDeployment().GetName() == "" ||
		exclusion.GetDeployment().GetName() == m.deployment.GetName()
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
