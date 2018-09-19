package deploytime

import (
	"fmt"
	"sync"

	"github.com/gogo/protobuf/proto"
	policyDatastore "github.com/stackrox/rox/central/policy/datastore"
	"github.com/stackrox/rox/generated/api/v1"
	deploymentMatcher "github.com/stackrox/rox/pkg/compiledpolicies/deployment/matcher"
)

type setImpl struct {
	lock sync.RWMutex

	policyIDToPolicy  map[string]*v1.Policy
	policyIDToMatcher map[string]deploymentMatcher.Matcher

	runtimePolicyIDToMatcher map[string]deploymentMatcher.Matcher

	policyStore policyDatastore.DataStore
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

// ForEach runs the given function on all present policies.
func (p *setImpl) ForEach(fe func(*v1.Policy, deploymentMatcher.Matcher) error, runtime bool) error {
	p.lock.RLock()
	defer p.lock.RUnlock()

	if runtime {
		for id, matcher := range p.runtimePolicyIDToMatcher {
			if err := fe(p.policyIDToPolicy[id], matcher); err != nil {
				return err
			}
		}
	} else {
		for id, matcher := range p.policyIDToMatcher {
			if err := fe(p.policyIDToPolicy[id], matcher); err != nil {
				return err
			}
		}
	}
	return nil
}

// UpsertPolicy adds or updates a policy in the set.
func (p *setImpl) UpsertPolicy(policy *v1.Policy) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	cloned := proto.Clone(policy).(*v1.Policy)

	matcher, err := deploymentMatcher.Compile(cloned)
	if err != nil {
		return err
	}

	p.policyIDToPolicy[cloned.GetId()] = cloned
	if cloned.GetLifecycleStage() == v1.LifecycleStage_RUN_TIME {
		p.runtimePolicyIDToMatcher[cloned.GetId()] = matcher
	} else {
		p.policyIDToMatcher[cloned.GetId()] = matcher
	}
	return nil
}

// RemovePolicy removes a policy from the set.
func (p *setImpl) RemovePolicy(policyID string) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	if _, exists := p.policyIDToPolicy[policyID]; exists {
		delete(p.policyIDToPolicy, policyID)
		if _, exists := p.policyIDToMatcher[policyID]; exists {
			delete(p.policyIDToMatcher, policyID)
		} else if _, exists := p.runtimePolicyIDToMatcher[policyID]; exists {
			delete(p.runtimePolicyIDToMatcher, policyID)
		}
	}

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
