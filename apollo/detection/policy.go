package detection

import (
	"fmt"

	"bitbucket.org/stack-rox/apollo/apollo/detection/processors"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/uuid"
	"github.com/golang/protobuf/ptypes"

	// Initialize each category.
	_ "bitbucket.org/stack-rox/apollo/apollo/detection/configuration_processor"
	_ "bitbucket.org/stack-rox/apollo/apollo/detection/image_processor"
	_ "bitbucket.org/stack-rox/apollo/apollo/detection/privilege_processor"
)

type policyWrapper struct {
	*v1.Policy
	compiled []processors.CompiledPolicy
}

func (d *Detector) initializePolicies() error {
	d.policies = make(map[string]*policyWrapper)

	policies, err := d.database.GetPolicies(&v1.GetPoliciesRequest{})
	if err != nil {
		return err
	}

	for _, policy := range policies {
		if err := d.addPolicy(policy); err != nil {
			return fmt.Errorf("policy %s: %s", policy.GetName(), err)
		}
	}
	return nil
}

func (d *Detector) addPolicy(policy *v1.Policy) (err error) {
	var p *policyWrapper
	if p, err = newPolicyWrapper(policy); err != nil {
		return err
	}

	d.policies[policy.GetName()] = p
	return
}

func newPolicyWrapper(policy *v1.Policy) (*policyWrapper, error) {
	p := &policyWrapper{
		Policy: policy,
	}

	for _, c := range policy.GetCategories() {
		compiler, ok := processors.PolicyCategoryCompiler[c]
		if !ok {
			return nil, fmt.Errorf("policy compiler not found for %s", c)
		}
		compiled, err := compiler(policy)
		if err != nil {
			return nil, fmt.Errorf("policy Category %s failed to compile: %s", c, err)
		}

		p.compiled = append(p.compiled, compiled)
	}

	return p, nil
}

// UpdatePolicy updates the current policy in a threadsafe manner.
func (d *Detector) UpdatePolicy(policy *v1.Policy) error {
	d.policyMutex.Lock()
	defer d.policyMutex.Unlock()
	return d.addPolicy(policy)
}

// RemovePolicy removes the policy specified by name in a threadsafe manner.
func (d *Detector) RemovePolicy(name string) {
	d.policyMutex.Lock()
	defer d.policyMutex.Unlock()
	delete(d.policies, name)
}

func (d *Detector) matchPolicy(deployment *v1.Deployment, p *policyWrapper) *v1.Alert {
	var violations []*v1.Alert_Violation

	// each container is considered independently.
	for _, c := range deployment.GetContainers() {
		violations = append(violations, p.Match(deployment, c)...)
	}

	if len(violations) == 0 {
		return nil
	}

	return &v1.Alert{
		Id:         uuid.NewV4().String(),
		Deployment: deployment,
		Policy:     p.Policy,
		Violations: violations,
		Time:       ptypes.TimestampNow(),
	}
}

func (p *policyWrapper) Match(deployment *v1.Deployment, container *v1.Container) (violations []*v1.Alert_Violation) {
	for _, c := range p.compiled {
		vs := c.Match(deployment, container)

		// All policy categories must match, otherwise no violations are returned
		if len(vs) == 0 {
			return []*v1.Alert_Violation{}
		}

		violations = append(violations, vs...)
	}

	return
}
