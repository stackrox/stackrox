package matcher

import (
	"fmt"

	"bitbucket.org/stack-rox/apollo/central/detection/processors"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"

	// Initialize each category.
	_ "bitbucket.org/stack-rox/apollo/central/detection/configuration_processor"
	_ "bitbucket.org/stack-rox/apollo/central/detection/image_processor"
	_ "bitbucket.org/stack-rox/apollo/central/detection/privilege_processor"
)

// Policy wraps the original v1 Policy and compiled policy suitable for computing matches against deployments.
type Policy struct {
	*v1.Policy
	compiled []processors.CompiledPolicy
}

// New returns a new policy.
func New(policy *v1.Policy) (*Policy, error) {
	p := &Policy{
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

// Match returns violations if deployment violates the policy.
func (p *Policy) Match(deployment *v1.Deployment) (violations []*v1.Alert_Violation) {
	// each container is considered independently.
	for _, c := range deployment.GetContainers() {
		violations = append(violations, p.matchContainer(deployment, c)...)
	}

	return
}

func (p *Policy) matchContainer(deployment *v1.Deployment, container *v1.Container) (violations []*v1.Alert_Violation) {
	for _, c := range p.compiled {
		vs := c.Match(deployment, container)

		// All policy categories must match, otherwise no violations are returned
		if len(vs) == 0 {
			return nil
		}

		violations = append(violations, vs...)
	}

	return
}

// ShouldProcess returns true if the policy is enabled and either the policy does not have scope constraints, or the deployment matches the scope.
func (p *Policy) ShouldProcess(deployment *v1.Deployment) bool {
	if p.Disabled {
		return false
	}

	if len(p.GetScope()) == 0 {
		return true
	}

	for _, s := range p.GetScope() {
		if p.withinScope(s, deployment) {
			return true
		}
	}

	return false
}

func (p *Policy) withinScope(scope *v1.Policy_Scope, deployment *v1.Deployment) bool {
	if cluster := scope.GetCluster(); cluster != "" && deployment.GetClusterId() != cluster {
		return false
	}

	if namespace := scope.GetNamespace(); namespace != "" && deployment.GetNamespace() != namespace {
		return false
	}

	if label := scope.GetLabel(); label != nil && deployment.GetLabels()[label.GetKey()] != label.GetValue() {
		return false
	}

	return true
}

// GetEnforcementAction returns the appropriate enforcement action for deployment.
func (p *Policy) GetEnforcementAction(deployment *v1.Deployment, action v1.ResourceAction) (enforcement v1.EnforcementAction, message string) {
	if !p.GetEnforce() {
		return
	}

	if action != v1.ResourceAction_CREATE_RESOURCE {
		return
	}

	if deployment.GetType() == "Global" || deployment.GetType() == "DaemonSet" {
		return
	}

	return v1.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT, fmt.Sprintf("Deployment %s scaled to 0 replicas in response to policy violation", deployment.GetName())
}
