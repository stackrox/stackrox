package image

import (
	policyDatastore "github.com/stackrox/rox/central/policy/datastore"
	"github.com/stackrox/rox/generated/api/v1"
	imageMatcher "github.com/stackrox/rox/pkg/compiledpolicies/image/matcher"
)

// PolicySet is a set of build time policies.
type PolicySet interface {
	ForOne(pID string, fe func(*v1.Policy, imageMatcher.Matcher) error) error
	ForEach(func(*v1.Policy, imageMatcher.Matcher) error) error

	UpsertPolicy(*v1.Policy) error
	RemovePolicy(policyID string) error

	RemoveNotifier(notifierID string) error
}

// NewPolicySet returns a new instance of a PolicySet.
func NewPolicySet(store policyDatastore.DataStore) PolicySet {
	return &setImpl{
		policyIDToPolicy:  make(map[string]*v1.Policy),
		policyIDToMatcher: make(map[string]imageMatcher.Matcher),
		policyStore:       store,
	}
}
