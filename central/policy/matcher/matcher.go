package matcher

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/kubernetes"
)

// Matcher provides functionality to evaluate whether or not policies are applicable to an entity.
type Matcher interface {
	FilterApplicablePolicies(ctx context.Context, policies []*storage.Policy) (applicable []*storage.Policy, notApplicable []*storage.Policy)
	IsPolicyApplicable(ctx context.Context, policy *storage.Policy) bool
}

// appliesToDeployments reports whether the exclusion can take a deployment out of policy scope.
// A deployment name/scope exclusion and a workload-kind exclusion both can. An image-only exclusion
// cannot: its nil deployment scope matches every deployment and would make the policy look
// inapplicable everywhere.
func appliesToDeployments(exclusion *storage.Exclusion) bool {
	return exclusion.GetDeployment() != nil || exclusion.GetExcludeByKind() != nil
}

// kindExclusionMatches reports whether a kind exclusion excludes this deployment.
// The feature flag gates it. A disabled flag leaves the policy applicable, matching detection.
func kindExclusionMatches(exclusion *storage.Exclusion, deploymentType string) bool {
	if !features.PolicyWorkloadKindExclusion.Enabled() {
		return false
	}
	for _, kind := range exclusion.GetExcludeByKind().GetKinds() {
		switch kind {
		case storage.Exclusion_CRON_JOB:
			if deploymentType == kubernetes.CronJob {
				return true
			}
		case storage.Exclusion_JOB:
			if deploymentType == kubernetes.Job {
				return true
			}
		}
	}
	return false
}
