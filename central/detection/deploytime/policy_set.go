package deploytime

import (
	"github.com/stackrox/rox/generated/api/v1"
	deploymentMatcher "github.com/stackrox/rox/pkg/compiledpolicies/deployment/matcher"
)

// PolicySet is a set of build time policies.
//go:generate mockery -name=PolicySet
type PolicySet interface {
	ForOne(string, func(*v1.Policy, deploymentMatcher.Matcher) error) error
	ForEach(func(*v1.Policy, deploymentMatcher.Matcher) error) error

	UpsertPolicy(*v1.Policy) error
	RemovePolicy(policyID string) error
	RemoveNotifier(notifierID string) error
}

// NewPolicySet returns a new instance of a PolicySet.
func NewPolicySet() PolicySet {
	return &setImpl{
		policyIDToPolicy: make(map[string]*v1.Policy),
		policyToMatcher:  make(map[*v1.Policy]deploymentMatcher.Matcher),
	}
}
