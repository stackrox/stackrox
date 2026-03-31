package matcher

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// Matcher provides functionality to evaluate whether or not policies are applicable to an entity.
type Matcher interface {
	FilterApplicablePolicies(ctx context.Context, policies []*storage.Policy) (applicable []*storage.Policy, notApplicable []*storage.Policy)
	IsPolicyApplicable(ctx context.Context, policy *storage.Policy) bool
}
