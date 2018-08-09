package detection

import (
	"github.com/stackrox/rox/pkg/compiledpolicies"
)

// UpdatePolicy updates the current policy in a threadsafe manner.
func (d *detectorImpl) UpdatePolicy(policy compiledpolicies.DeploymentMatcher) {
	d.policyMutex.Lock()
	defer d.policyMutex.Unlock()

	d.addPolicy(policy)
	d.notificationProcessor.UpdatePolicy(policy.GetProto())
}

// RemovePolicy removes the policy specified by id in a threadsafe manner.
func (d *detectorImpl) RemovePolicy(id string) {
	d.policyMutex.Lock()
	defer d.policyMutex.Unlock()

	p, ok := d.policies[id]
	if ok {
		p.GetProto().Disabled = true
		go d.reprocessPolicy(p)
		delete(d.policies, id)
		d.notificationProcessor.RemovePolicy(p.GetProto())
	}
}

// RemoveNotifier updates all policies that use provided notifier.
func (d *detectorImpl) RemoveNotifier(id string) {
	d.policyMutex.Lock()
	defer d.policyMutex.Unlock()

	for _, p := range d.policies {
		filtered := p.GetProto().GetNotifiers()[:0]

		for _, n := range p.GetProto().GetNotifiers() {
			if n != id {
				filtered = append(filtered, n)
			}
		}

		if len(p.GetProto().GetNotifiers()) != len(filtered) {
			p.GetProto().Notifiers = filtered
			if err := d.policyStorage.UpdatePolicy(p.GetProto()); err != nil {
				logger.Errorf("unable to update policy: %s", err)
			}
		}
	}
}

func (d *detectorImpl) initializePolicies() error {
	d.policies = make(map[string]compiledpolicies.DeploymentMatcher)

	policies, err := d.policyStorage.GetPolicies()
	if err != nil {
		return err
	}

	for _, policy := range policies {
		matcherPolicy, err := compiledpolicies.New(policy)
		if err != nil {
			logger.Errorf("policy %s: %s", policy.GetId(), err)
			continue
		}
		d.addPolicy(matcherPolicy)
	}
	return nil
}

func (d *detectorImpl) addPolicy(policy compiledpolicies.DeploymentMatcher) {
	d.policies[policy.GetProto().GetId()] = policy
	go d.reprocessPolicy(policy)
	return
}

func (d *detectorImpl) getCurrentPolicies() (policies []compiledpolicies.DeploymentMatcher) {
	d.policyMutex.RLock()
	defer d.policyMutex.RUnlock()

	for _, p := range d.policies {
		policies = append(policies, p)
	}

	return
}
