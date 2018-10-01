package deployment

import (
	policyDatastore "github.com/stackrox/rox/central/policy/datastore"
	"github.com/stackrox/rox/central/searchbasedpolicies"
	"github.com/stackrox/rox/generated/api/v1"
	deploymentMatcher "github.com/stackrox/rox/pkg/compiledpolicies/deployment/matcher"
	"github.com/stackrox/rox/pkg/compiledpolicies/deployment/predicate"
)

// PolicySet is a set of policies.
type PolicySet interface {
	ForOne(string, func(*v1.Policy, deploymentMatcher.Matcher) error) error
	ForOneSearchBased(policyID string, f func(*v1.Policy, searchbasedpolicies.Matcher, predicate.Predicate) error) error

	ForEach(fe func(*v1.Policy, deploymentMatcher.Matcher) error) error
	ForEachSearchBased(func(*v1.Policy, searchbasedpolicies.Matcher, predicate.Predicate) error) error

	UpsertPolicy(*v1.Policy) error
	RemovePolicy(policyID string) error
	RemoveNotifier(notifierID string) error
}

// NewPolicySet returns a new instance of a PolicySet.
func NewPolicySet(store policyDatastore.DataStore) PolicySet {
	return &setImpl{
		policyIDToPolicy:             make(map[string]*v1.Policy),
		policyIDToMatcher:            make(map[string]deploymentMatcher.Matcher),
		policyIDToSearchBasedMatcher: make(map[string]predicatedMatcher),
		policyStore:                  store,
	}
}
