package predicate

import (
	"github.com/stackrox/rox/generated/api/v1"
)

func init() {
	compilers = append(compilers, newDisabledPredicate)
}

func newDisabledPredicate(policy *v1.Policy) (Predicate, error) {
	if !policy.GetDisabled() {
		return nil, nil
	}
	return shouldProcess, nil
}

// If the policy is disabled, we create a predicate that always returns false.
func shouldProcess(*v1.Deployment) bool {
	return false
}
