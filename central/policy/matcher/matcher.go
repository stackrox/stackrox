package matcher

import "github.com/stackrox/rox/generated/storage"

// Matcher provides functionality to evaluate whether or not policies are applicable to an entity.
type Matcher interface {
	FilterApplicablePolicies(policies []*storage.Policy) (applicable []*storage.Policy, notApplicable []*storage.Policy)
	IsPolicyApplicable(policy *storage.Policy) bool
}

// appliesToDeployments reports whether the exclusion has a deployment part. Image-only exclusions
// are applied to images, so they must not exclude deployments, namespaces, or clusters. Without this
// check, the nil deployment scope matches everything and the policy looks inapplicable everywhere.
func appliesToDeployments(exclusion *storage.Exclusion) bool {
	return exclusion.GetDeployment() != nil
}
