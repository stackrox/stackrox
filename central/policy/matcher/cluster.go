package matcher

import (
	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/scopecomp"
	"github.com/stackrox/stackrox/pkg/utils"
)

type clusterMatcher struct {
	cluster    *storage.Cluster
	namespaces []*storage.NamespaceMetadata
}

// NewClusterMatcher creates a new policy matcher for cluster data.
func NewClusterMatcher(cluster *storage.Cluster, namespaces []*storage.NamespaceMetadata) Matcher {
	return &clusterMatcher{
		cluster:    cluster,
		namespaces: namespaces,
	}
}

// FilterApplicablePolicies filters incoming policies into policies that apply to cluster and policies that do not apply to cluster
func (m *clusterMatcher) FilterApplicablePolicies(policies []*storage.Policy) ([]*storage.Policy, []*storage.Policy) {
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

// IsPolicyApplicable returns true if the policy is applicable to cluster
func (m *clusterMatcher) IsPolicyApplicable(policy *storage.Policy) bool {
	return !policy.GetDisabled() && !m.anyExclusionMatches(policy.GetExclusions()) && m.anyScopeMatches(policy.GetScope())
}

func (m *clusterMatcher) anyExclusionMatches(exclusions []*storage.Exclusion) bool {
	for _, exclusion := range exclusions {
		if m.exclusionMatches(exclusion) {
			return true
		}
	}
	return false
}

func (m *clusterMatcher) exclusionMatches(exclusion *storage.Exclusion) bool {
	cs, err := scopecomp.CompileScope(exclusion.GetDeployment().GetScope())
	if err != nil {
		utils.Should(errors.Wrap(err, "could not compile excluded scopes"))
		return false
	}

	if !cs.MatchesCluster(m.cluster.GetId()) {
		return false
	}

	if exclusion.GetDeployment().GetScope().GetNamespace() == "" {
		return true
	}

	for _, namespace := range m.namespaces {
		if !cs.MatchesNamespace(namespace.GetName()) {
			return false
		}
	}
	return true
}

func (m *clusterMatcher) anyScopeMatches(scopes []*storage.Scope) bool {
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

func (m *clusterMatcher) scopeMatches(scope *storage.Scope) bool {
	cs, err := scopecomp.CompileScope(scope)
	if err != nil {
		utils.Should(errors.Wrap(err, "could not compile scope"))
		return false
	}

	if !cs.MatchesCluster(m.cluster.GetId()) {
		return false
	}

	if scope.GetNamespace() == "" {
		return true
	}

	for _, namespace := range m.namespaces {
		if cs.MatchesNamespace(namespace.GetName()) {
			return true
		}
	}
	return false
}
