package deployment

import (
	"fmt"
	"sync"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/central/deployment/index/mappings"
	policyDatastore "github.com/stackrox/rox/central/policy/datastore"
	"github.com/stackrox/rox/central/searchbasedpolicies"
	"github.com/stackrox/rox/central/searchbasedpolicies/matcher"
	"github.com/stackrox/rox/generated/api/v1"
	deploymentMatcher "github.com/stackrox/rox/pkg/compiledpolicies/deployment/matcher"
	"github.com/stackrox/rox/pkg/compiledpolicies/deployment/predicate"
	"github.com/stackrox/rox/pkg/policies"
)

type predicatedMatcher struct {
	m searchbasedpolicies.Matcher
	p predicate.Predicate
}

type setImpl struct {
	lock sync.RWMutex

	policyIDToPolicy  map[string]*v1.Policy
	policyIDToMatcher map[string]deploymentMatcher.Matcher
	policyStore       policyDatastore.DataStore

	policyIDToSearchBasedMatcher map[string]predicatedMatcher
}

func (p *setImpl) ForEachSearchBased(f func(*v1.Policy, searchbasedpolicies.Matcher, predicate.Predicate) error) error {
	p.lock.RLock()
	defer p.lock.RUnlock()
	for id, matcher := range p.policyIDToSearchBasedMatcher {
		if err := f(p.policyIDToPolicy[id], matcher.m, matcher.p); err != nil {
			return err
		}
	}
	return nil
}

// ForOne runs the given function on the policy matching the id if it exists.
func (p *setImpl) ForOne(pID string, fe func(*v1.Policy, deploymentMatcher.Matcher) error) error {
	p.lock.RLock()
	defer p.lock.RUnlock()

	if policy, exists := p.policyIDToPolicy[pID]; exists {
		return fe(policy, p.policyIDToMatcher[pID])
	}
	return fmt.Errorf("policy with ID not found in set: %s", pID)
}

func (p *setImpl) ForOneSearchBased(pID string, f func(*v1.Policy, searchbasedpolicies.Matcher, predicate.Predicate) error) error {
	p.lock.RLock()
	defer p.lock.RUnlock()

	if policy, exists := p.policyIDToPolicy[pID]; exists {
		predicatedMatcher := p.policyIDToSearchBasedMatcher[pID]
		return f(policy, predicatedMatcher.m, predicatedMatcher.p)
	}
	return fmt.Errorf("policy with ID not found in set: %s", pID)
}

// ForEach runs the given function on all present policies.
func (p *setImpl) ForEach(fe func(*v1.Policy, deploymentMatcher.Matcher) error) error {
	p.lock.RLock()
	defer p.lock.RUnlock()

	for id, matcher := range p.policyIDToMatcher {
		if err := fe(p.policyIDToPolicy[id], matcher); err != nil {
			return err
		}
	}
	return nil
}

// UpsertPolicy adds or updates a policy in the set.
func (p *setImpl) UpsertPolicy(policy *v1.Policy) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	cloned := proto.Clone(policy).(*v1.Policy)

	// TODO(viswa): This breaks the abstraction, but leaving it like this to facilitate an easy
	// transition of runtime policies to search.
	if policies.AppliesAtRunTime(policy) {
		m, err := deploymentMatcher.Compile(cloned)
		if err != nil {
			return err
		}
		p.policyIDToMatcher[cloned.GetId()] = m
	} else {
		searchBasedMatcher, err := matcher.ForPolicy(cloned, mappings.OptionsMap)
		if err != nil {
			return err
		}
		pred, err := predicate.Compile(cloned)
		if err != nil {
			return err
		}
		p.policyIDToSearchBasedMatcher[cloned.GetId()] = predicatedMatcher{m: searchBasedMatcher, p: pred}
	}

	p.policyIDToPolicy[cloned.GetId()] = cloned
	return nil
}

// RemovePolicy removes a policy from the set.
func (p *setImpl) RemovePolicy(policyID string) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	delete(p.policyIDToPolicy, policyID)
	delete(p.policyIDToMatcher, policyID)
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
