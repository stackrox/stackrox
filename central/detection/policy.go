package detection

import (
	"bitbucket.org/stack-rox/apollo/central/detection/matcher"
)

// UpdatePolicy updates the current policy in a threadsafe manner.
func (d *detectorImpl) UpdatePolicy(policy *matcher.Policy) {
	d.policyMutex.Lock()
	defer d.policyMutex.Unlock()

	d.addPolicy(policy)
	d.notificationProcessor.UpdatePolicy(policy.Policy)
}

// RemovePolicy removes the policy specified by id in a threadsafe manner.
func (d *detectorImpl) RemovePolicy(id string) {
	d.policyMutex.Lock()
	defer d.policyMutex.Unlock()

	p, ok := d.policies[id]
	if ok {
		p.Disabled = true
		go d.reprocessPolicy(p)
		delete(d.policies, id)
		d.notificationProcessor.RemovePolicy(p.Policy)
	}
}

// RemoveNotifier updates all policies that use provided notifier.
func (d *detectorImpl) RemoveNotifier(id string) {
	d.policyMutex.Lock()
	defer d.policyMutex.Unlock()

	for _, p := range d.policies {
		filtered := p.GetNotifiers()[:0]

		for _, n := range p.GetNotifiers() {
			if n != id {
				filtered = append(filtered, n)
			}
		}

		if len(p.GetNotifiers()) != len(filtered) {
			p.Notifiers = filtered
			if err := d.policyStorage.UpdatePolicy(p.Policy); err != nil {
				logger.Errorf("unable to update policy: %s", err)
			}
		}
	}
}

func (d *detectorImpl) initializePolicies() error {
	d.policies = make(map[string]*matcher.Policy)

	policies, err := d.policyStorage.GetPolicies()
	if err != nil {
		return err
	}

	for _, policy := range policies {
		matcherPolicy, err := matcher.New(policy)
		if err != nil {
			logger.Errorf("policy %s: %s", policy.GetId(), err)
			continue
		}
		d.addPolicy(matcherPolicy)
	}
	return nil
}

func (d *detectorImpl) addPolicy(policy *matcher.Policy) {
	d.policies[policy.GetId()] = policy
	go d.reprocessPolicy(policy)
	return
}

func (d *detectorImpl) getCurrentPolicies() (policies []*matcher.Policy) {
	d.policyMutex.RLock()
	defer d.policyMutex.RUnlock()

	for _, p := range d.policies {
		policies = append(policies, p)
	}

	return
}
