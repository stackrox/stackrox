package detection

import (
	"bitbucket.org/stack-rox/apollo/central/detection/matcher"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
)

func (d *Detector) initializePolicies() error {
	d.policies = make(map[string]*matcher.Policy)

	policies, err := d.database.GetPolicies(&v1.GetPoliciesRequest{})
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

func (d *Detector) addPolicy(policy *matcher.Policy) {
	d.policies[policy.GetId()] = policy
	go d.reprocessPolicy(policy)
	return
}

// UpdatePolicy updates the current policy in a threadsafe manner.
func (d *Detector) UpdatePolicy(policy *matcher.Policy) {
	d.policyMutex.Lock()
	defer d.policyMutex.Unlock()
	d.addPolicy(policy)
}

// RemovePolicy removes the policy specified by id in a threadsafe manner.
func (d *Detector) RemovePolicy(id string) {
	d.policyMutex.Lock()
	defer d.policyMutex.Unlock()
	p, ok := d.policies[id]
	if ok {
		p.Disabled = true
		go d.reprocessPolicy(p)
		delete(d.policies, id)
	}
}

func (d *Detector) getCurrentPolicies() (policies []*matcher.Policy) {
	d.policyMutex.Lock()
	defer d.policyMutex.Unlock()

	for _, p := range d.policies {
		policies = append(policies, p)
	}

	return
}
