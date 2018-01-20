package detection

import (
	"fmt"

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
		if err := d.addPolicy(policy); err != nil {
			return fmt.Errorf("policy %s: %s", policy.GetId(), err)
		}
	}
	return nil
}

func (d *Detector) addPolicy(policy *v1.Policy) (err error) {
	var p *matcher.Policy
	if p, err = matcher.New(policy); err != nil {
		return err
	}

	d.policies[policy.GetId()] = p
	go d.reprocessPolicy(p)
	return
}

// UpdatePolicy updates the current policy in a threadsafe manner.
func (d *Detector) UpdatePolicy(policy *v1.Policy) error {
	d.policyMutex.Lock()
	defer d.policyMutex.Unlock()

	return d.addPolicy(policy)
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
