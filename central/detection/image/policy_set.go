package image

import (
	policyDatastore "github.com/stackrox/rox/central/policy/datastore"
	"github.com/stackrox/rox/central/searchbasedpolicies"
	"github.com/stackrox/rox/generated/api/v1"
)

// PolicySet is a set of build time policies.
type PolicySet interface {
	ForEach(func(*v1.Policy, searchbasedpolicies.Matcher) error) error

	UpsertPolicy(*v1.Policy) error
	RemovePolicy(policyID string) error

	RemoveNotifier(notifierID string) error
}

// NewPolicySet returns a new instance of a PolicySet.
func NewPolicySet(store policyDatastore.DataStore) PolicySet {
	return &setImpl{
		policyIDToPolicy:  make(map[string]*v1.Policy),
		policyIDToMatcher: make(map[string]searchbasedpolicies.Matcher),
		policyStore:       store,
	}
}
