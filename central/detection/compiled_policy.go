package detection

import (
	"github.com/stackrox/rox/central/searchbasedpolicies"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/policies"
)

// CompiledPolicy is a compiled policy, which means it has a generated matcher and predicate function.
type CompiledPolicy interface {
	Policy() *storage.Policy
	Matcher() searchbasedpolicies.Matcher
	Predicate
}

// NewCompiledPolicy creates and returns a compiled policy from the policy and matcher.
func NewCompiledPolicy(policy *storage.Policy, matcher searchbasedpolicies.Matcher) (CompiledPolicy, error) {
	compiled := &compiledPolicy{
		policy:  policy,
		matcher: matcher,
	}

	if policies.AppliesAtDeployTime(policy) || policies.AppliesAtRunTime(policy) {
		compiled.predicates = append(compiled.predicates, &deploymentPredicate{policy: policy})
	}
	if policies.AppliesAtBuildTime(policy) {
		compiled.predicates = append(compiled.predicates, &imagePredicate{
			policy: policy,
		})
	}
	return compiled, nil
}

// Top level compiled Policy.
type compiledPolicy struct {
	policy     *storage.Policy
	matcher    searchbasedpolicies.Matcher
	predicates []Predicate
}

// Policy returns the policy that was compiled.
func (cp *compiledPolicy) Policy() *storage.Policy {
	return cp.policy
}

// Matcher returns the matcher constructed for the policy.
func (cp *compiledPolicy) Matcher() searchbasedpolicies.Matcher {
	return cp.matcher
}

// IsEnabledAndAppliesTo returns if the compiled policy applies to the input object.
func (cp *compiledPolicy) IsEnabledAndAppliesTo(input interface{}) bool {
	for _, predicate := range cp.predicates {
		if predicate.IsEnabledAndAppliesTo(input) {
			return true
		}
	}
	return false
}

// Predicate says whether or not a compiled policy applies to an object.
type Predicate interface {
	IsEnabledAndAppliesTo(interface{}) bool
}

// Predicate for deployments.
type deploymentPredicate struct {
	policy *storage.Policy
}

func (cp *deploymentPredicate) IsEnabledAndAppliesTo(input interface{}) bool {
	if cp.policy.GetDisabled() {
		return false
	}
	deployment, isDeployment := input.(*storage.Deployment)
	if !isDeployment {
		return false
	}
	return !matchesDeploymentWhitelists(deployment, cp.policy)
}

// Predicate for images.
type imagePredicate struct {
	policy *storage.Policy
}

func (cp *imagePredicate) IsEnabledAndAppliesTo(input interface{}) bool {
	if cp.policy.GetDisabled() {
		return false
	}
	image, isImage := input.(*storage.Image)
	if !isImage {
		return false
	}
	return !matchesImageWhitelist(image.GetName().GetFullName(), cp.policy)
}
