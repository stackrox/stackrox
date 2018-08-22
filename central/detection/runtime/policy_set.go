package runtime

import (
	"github.com/stackrox/rox/generated/api/v1"
	containerMatcher "github.com/stackrox/rox/pkg/compiledpolicies/container/matcher"
)

// PolicySet is a set of build time policies.
type PolicySet interface {
	ForOne(pID string, fe func(*v1.Policy, containerMatcher.Matcher) error) error
	ForEach(fe func(*v1.Policy, containerMatcher.Matcher) error) error

	UpsertPolicy(*v1.Policy) error
	RemovePolicy(policyID string) error
	RemoveNotifier(notifierID string) error
}

// NewPolicySet returns a new instance of a PolicySet.
func NewPolicySet() PolicySet {
	return &setImpl{
		policyIDToPolicy: make(map[string]*v1.Policy),
		policyToMatcher:  make(map[*v1.Policy]containerMatcher.Matcher),
	}
}
