package matcher

import (
	"context"

	"github.com/pkg/errors"
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	namespaceDataStore "github.com/stackrox/rox/central/namespace/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/scopecomp"
	"github.com/stackrox/rox/pkg/utils"
)

type deploymentMatcher struct {
	deployment        *storage.Deployment
	clusterProvider   scopecomp.ClusterLabelProvider
	namespaceProvider scopecomp.NamespaceLabelProvider
}

// NewDeploymentMatcher creates a new policy matcher for deployment data.
func NewDeploymentMatcher(deployment *storage.Deployment, clusterDS clusterDataStore.DataStore, namespaceDS namespaceDataStore.DataStore) Matcher {
	return &deploymentMatcher{
		deployment:        deployment,
		clusterProvider:   clusterDS,
		namespaceProvider: namespaceDS,
	}
}

// FilterApplicablePolicies filters incoming policies into policies that apply to deployment and policies that do not apply to deployment
func (m *deploymentMatcher) FilterApplicablePolicies(ctx context.Context, policies []*storage.Policy) ([]*storage.Policy, []*storage.Policy) {
	applicable := make([]*storage.Policy, 0, len(policies)/2)
	notApplicable := make([]*storage.Policy, 0, len(policies)/2)

	for _, policy := range policies {
		if m.IsPolicyApplicable(ctx, policy) {
			applicable = append(applicable, policy)
		} else {
			notApplicable = append(notApplicable, policy)
		}
	}
	return applicable, notApplicable
}

// IsPolicyApplicable returns true if the policy is applicable to deployment
func (m *deploymentMatcher) IsPolicyApplicable(ctx context.Context, policy *storage.Policy) bool {
	return !policy.GetDisabled() && !m.anyExclusionMatches(ctx, policy.GetExclusions()) && m.anyScopeMatches(ctx, policy.GetScope())
}

func (m *deploymentMatcher) anyExclusionMatches(ctx context.Context, exclusions []*storage.Exclusion) bool {
	for _, exclusion := range exclusions {
		if m.exclusionMatches(ctx, exclusion) {
			return true
		}
	}
	return false
}

func (m *deploymentMatcher) exclusionMatches(ctx context.Context, exclusion *storage.Exclusion) bool {
	// If excluded scope does not match the deployment then no need to check for deployment name
	if !m.scopeMatches(ctx, exclusion.GetDeployment().GetScope()) {
		return false
	}

	// If scope of exclusions matches, the deployment is excluded if,
	// - Deployment name is not set, or
	// - Deployment name matches,
	return exclusion.GetDeployment().GetName() == "" ||
		exclusion.GetDeployment().GetName() == m.deployment.GetName()
}

func (m *deploymentMatcher) anyScopeMatches(ctx context.Context, scopes []*storage.Scope) bool {
	if len(scopes) == 0 {
		return true
	}

	for _, scope := range scopes {
		if m.scopeMatches(ctx, scope) {
			return true
		}
	}
	return false
}

func (m *deploymentMatcher) scopeMatches(ctx context.Context, scope *storage.Scope) bool {
	cs, err := scopecomp.CompileScope(scope, m.clusterProvider, m.namespaceProvider)
	if err != nil {
		utils.Should(errors.Wrap(err, "could not compile scope"))
		return false
	}

	return cs.MatchesDeployment(ctx, m.deployment)
}
