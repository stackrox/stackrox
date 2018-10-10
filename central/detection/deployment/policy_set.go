package deployment

import (
	policyDatastore "github.com/stackrox/rox/central/policy/datastore"
	processDataStore "github.com/stackrox/rox/central/processindicator/datastore"
	"github.com/stackrox/rox/central/searchbasedpolicies"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/compiledpolicies/deployment/predicate"
)

// PolicySet is a set of policies.
type PolicySet interface {
	ForOne(policyID string, f func(*v1.Policy, searchbasedpolicies.Matcher, predicate.Predicate) error) error
	ForEach(func(*v1.Policy, searchbasedpolicies.Matcher, predicate.Predicate) error) error

	UpsertPolicy(*v1.Policy) error
	RemovePolicy(policyID string) error
	RemoveNotifier(notifierID string) error
}

// NewPolicySet returns a new instance of a PolicySet.
func NewPolicySet(store policyDatastore.DataStore, processStore processDataStore.DataStore) PolicySet {
	return &setImpl{
		policyIDToPolicy:             make(map[string]*v1.Policy),
		policyIDToSearchBasedMatcher: make(map[string]predicatedMatcher),
		policyStore:                  store,
		processStore:                 processStore,
	}
}
