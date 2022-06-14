package matcher

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/scopecomp"
	"github.com/stackrox/rox/pkg/utils"
)

type namespaceMatcher struct {
	namespace *storage.NamespaceMetadata
}

// NewNamespaceMatcher creates a new policy matcher for namespace data.
func NewNamespaceMatcher(namespace *storage.NamespaceMetadata) Matcher {
	return &namespaceMatcher{
		namespace: namespace,
	}
}

// FilterApplicablePolicies filters incoming policies into policies that apply to namespace and policies that do not apply to namespace
func (m *namespaceMatcher) FilterApplicablePolicies(policies []*storage.Policy) ([]*storage.Policy, []*storage.Policy) {
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

// IsPolicyApplicable returns true if the policy is applicable to namespace
func (m *namespaceMatcher) IsPolicyApplicable(policy *storage.Policy) bool {
	return !policy.GetDisabled() && !m.anyExclusionMatches(policy.GetExclusions()) && m.anyScopeMatches(policy.GetScope())
}

func (m *namespaceMatcher) anyExclusionMatches(exclusions []*storage.Exclusion) bool {
	for _, exclusion := range exclusions {
		if m.exclusionMatches(exclusion) {
			return true
		}
	}
	return false
}

func (m *namespaceMatcher) exclusionMatches(exclusion *storage.Exclusion) bool {
	return m.scopeMatches(exclusion.GetDeployment().GetScope())
}

func (m *namespaceMatcher) anyScopeMatches(scopes []*storage.Scope) bool {
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

func (m *namespaceMatcher) scopeMatches(scope *storage.Scope) bool {
	cs, err := scopecomp.CompileScope(scope)
	if err != nil {
		utils.Should(errors.Wrap(err, "could not compiled scope"))
		return false
	}

	if !cs.MatchesCluster(m.namespace.GetClusterId()) {
		return false
	}

	return cs.MatchesNamespace(m.namespace.GetName())
}
