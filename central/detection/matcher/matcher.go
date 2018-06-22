package matcher

import (
	"fmt"

	"bitbucket.org/stack-rox/apollo/central/detection/processors"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"

	// Initialize each category.
	_ "bitbucket.org/stack-rox/apollo/central/detection/configuration_processor"
	_ "bitbucket.org/stack-rox/apollo/central/detection/image_processor"
	_ "bitbucket.org/stack-rox/apollo/central/detection/privilege_processor"
	"bitbucket.org/stack-rox/apollo/pkg/images"
	"bitbucket.org/stack-rox/apollo/pkg/scopecomp"
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

	for _, c := range processors.PolicySegmentCompilers {
		compiled, err := c(policy)
		if err != nil {
			return nil, fmt.Errorf("policy %s failed to compile: %s", p, err)
		}
		p.compiled = append(p.compiled, compiled)
	}

	return p, nil
}

func (p *Policy) matchesContainerWhitelists(whitelists []*v1.Whitelist, container *v1.Container) bool {
	for _, whitelist := range p.GetWhitelists() {
		if p.matchesContainerWhitelist(whitelist.GetContainer(), container) {
			return true
		}
	}
	return false
}

// Match returns violations if deployment violates the policy.
func (p *Policy) Match(deployment *v1.Deployment) (violations []*v1.Alert_Violation, excluded *v1.DryRunResponse_Excluded) {
	for _, whitelist := range p.GetWhitelists() {
		if p.matchesDeploymentWhitelist(whitelist.GetDeployment(), deployment) {
			return nil, &v1.DryRunResponse_Excluded{
				Deployment: deployment.GetName(),
				Whitelist:  whitelist,
			}
		}
	}

	violations = p.matchDeployment(deployment)

	// each container is considered independently.
	for _, c := range deployment.GetContainers() {
		if p.matchesContainerWhitelists(p.GetWhitelists(), c) {
			continue
		}
		violations = append(violations, p.matchContainer(deployment, c)...)
	}

	return
}

func (p *Policy) matchDeployment(deployment *v1.Deployment) []*v1.Alert_Violation {
	var violations []*v1.Alert_Violation
	for _, c := range p.compiled {
		vs, exists := c.MatchDeployment(deployment)
		// All policy categories must match, otherwise no violations are returned
		if exists && len(vs) == 0 {
			return nil
		}
		violations = append(violations, vs...)
	}
	return violations
}

func (p *Policy) matchContainer(deployment *v1.Deployment, container *v1.Container) (violations []*v1.Alert_Violation) {
	for _, c := range p.compiled {
		vs, exists := c.MatchContainer(container)

		// All policy categories must match, otherwise no violations are returned
		if exists && len(vs) == 0 {
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
		if scopecomp.WithinScope(s, deployment) {
			return true
		}
	}

	return false
}

func (p *Policy) matchesDeploymentWhitelist(whitelist *v1.Whitelist_Deployment, deployment *v1.Deployment) bool {
	if whitelist == nil {
		return false
	}
	if whitelist.GetScope() != nil && !scopecomp.WithinScope(whitelist.GetScope(), deployment) {
		return false
	}
	if whitelist.GetName() != "" && whitelist.GetName() != deployment.GetName() {
		return false
	}

	return true
}

func (p *Policy) matchesContainerWhitelist(whitelist *v1.Whitelist_Container, container *v1.Container) bool {
	if whitelist == nil {
		return false
	}
	whitelistName := whitelist.GetImageName()
	containerName := container.GetImage().GetName()
	whitelistDigest := images.NewDigest(whitelistName.GetSha()).Digest()
	containerDigest := images.NewDigest(containerName.GetSha()).Digest()

	if whitelistName.GetSha() != "" && whitelistDigest != containerDigest {
		return false
	}
	if whitelistName.GetRegistry() != "" && whitelistName.GetRegistry() != containerName.GetRegistry() {
		return false
	}
	if whitelistName.GetRemote() != "" && whitelistName.GetRemote() != containerName.GetRemote() {
		return false
	}
	if whitelistName.GetTag() != "" && whitelistName.GetTag() != containerName.GetTag() {
		return false
	}
	return true
}
