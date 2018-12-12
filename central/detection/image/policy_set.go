package image

import (
	policyDatastore "github.com/stackrox/rox/central/policy/datastore"
	"github.com/stackrox/rox/central/searchbasedpolicies"
	"github.com/stackrox/rox/generated/storage"
)

// PolicySet is a set of build time policies.
type PolicySet interface {
	ForEach(func(*storage.Policy, searchbasedpolicies.Matcher) error) error

	UpsertPolicy(*storage.Policy) error
	RemovePolicy(policyID string) error

	RemoveNotifier(notifierID string) error
}

// NewPolicySet returns a new instance of a PolicySet.
func NewPolicySet(store policyDatastore.DataStore) PolicySet {
	return &setImpl{
		policyIDToPolicy:  make(map[string]*storage.Policy),
		policyIDToMatcher: make(map[string]searchbasedpolicies.Matcher),
		policyStore:       store,
	}
}
