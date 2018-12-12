package deployment

import (
	"fmt"
	"sync"
	"time"

	"github.com/stackrox/rox/central/deployment/index/mappings"
	"github.com/stackrox/rox/central/metrics"
	policyDatastore "github.com/stackrox/rox/central/policy/datastore"
	"github.com/stackrox/rox/central/processindicator/datastore"
	"github.com/stackrox/rox/central/searchbasedpolicies"
	"github.com/stackrox/rox/central/searchbasedpolicies/matcher"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/compiledpolicies/deployment/predicate"
	"github.com/stackrox/rox/pkg/protoutils"
)

type predicatedMatcher struct {
	m searchbasedpolicies.Matcher
	p predicate.Predicate
}

type setImpl struct {
	lock sync.RWMutex

	policyIDToPolicy map[string]*storage.Policy
	policyStore      policyDatastore.DataStore

	processStore datastore.DataStore

	policyIDToSearchBasedMatcher map[string]predicatedMatcher
}

func (p *setImpl) ForEach(f func(*storage.Policy, searchbasedpolicies.Matcher, predicate.Predicate) error) error {
	p.lock.RLock()
	defer p.lock.RUnlock()
	for id, matcher := range p.policyIDToSearchBasedMatcher {
		t := time.Now()
		if err := f(p.policyIDToPolicy[id], matcher.m, matcher.p); err != nil {
			return err
		}
		metrics.SetPolicyEvaluationDurationTime(t, p.policyIDToPolicy[id].GetName())
	}
	return nil
}

func (p *setImpl) ForOne(pID string, f func(*storage.Policy, searchbasedpolicies.Matcher, predicate.Predicate) error) error {
	p.lock.RLock()
	defer p.lock.RUnlock()

	if policy, exists := p.policyIDToPolicy[pID]; exists {
		predicatedMatcher := p.policyIDToSearchBasedMatcher[pID]
		return f(policy, predicatedMatcher.m, predicatedMatcher.p)
	}
	return fmt.Errorf("policy with ID not found in set: %s", pID)
}

// UpsertPolicy adds or updates a policy in the set.
func (p *setImpl) UpsertPolicy(policy *storage.Policy) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	cloned := protoutils.CloneStoragePolicy(policy)

	searchBasedMatcher, err := matcher.ForPolicy(cloned, mappings.OptionsMap, p.processStore)
	if err != nil {
		return err
	}
	pred, err := predicate.Compile(cloned)
	if err != nil {
		return err
	}
	p.policyIDToSearchBasedMatcher[cloned.GetId()] = predicatedMatcher{m: searchBasedMatcher, p: pred}

	p.policyIDToPolicy[cloned.GetId()] = cloned
	return nil
}

// RemovePolicy removes a policy from the set.
func (p *setImpl) RemovePolicy(policyID string) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	delete(p.policyIDToPolicy, policyID)
	delete(p.policyIDToSearchBasedMatcher, policyID)
	return nil
}

// RemoveNotifier removes a given notifier from any policies in the set that use it.
func (p *setImpl) RemoveNotifier(notifierID string) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	for _, policy := range p.policyIDToPolicy {
		filtered := policy.GetNotifiers()[:0]
		for _, n := range policy.GetNotifiers() {
			if n != notifierID {
				filtered = append(filtered, n)
			}
		}
		policy.Notifiers = filtered

		err := p.policyStore.UpdatePolicy(policy)
		if err != nil {
			return err
		}
	}

	return nil
}
