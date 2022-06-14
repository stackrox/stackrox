package matcher

import "github.com/stackrox/stackrox/generated/storage"

// Matcher provides functionality to evaluate whether or not policies are applicable to an entity.
type Matcher interface {
	FilterApplicablePolicies(policies []*storage.Policy) (applicable []*storage.Policy, notApplicable []*storage.Policy)
	IsPolicyApplicable(policy *storage.Policy) bool
}
