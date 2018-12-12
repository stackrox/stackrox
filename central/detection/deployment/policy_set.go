package deployment

import (
	policyDatastore "github.com/stackrox/rox/central/policy/datastore"
	processDataStore "github.com/stackrox/rox/central/processindicator/datastore"
	"github.com/stackrox/rox/central/searchbasedpolicies"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/compiledpolicies/deployment/predicate"
)

// PolicySet is a set of policies.
type PolicySet interface {
	ForOne(policyID string, f func(*storage.Policy, searchbasedpolicies.Matcher, predicate.Predicate) error) error
	ForEach(func(*storage.Policy, searchbasedpolicies.Matcher, predicate.Predicate) error) error

	UpsertPolicy(*storage.Policy) error
	RemovePolicy(policyID string) error
	RemoveNotifier(notifierID string) error
}

// NewPolicySet returns a new instance of a PolicySet.
func NewPolicySet(store policyDatastore.DataStore, processStore processDataStore.DataStore) PolicySet {
	return &setImpl{
		policyIDToPolicy:             make(map[string]*storage.Policy),
		policyIDToSearchBasedMatcher: make(map[string]predicatedMatcher),
		policyStore:                  store,
		processStore:                 processStore,
	}
}
