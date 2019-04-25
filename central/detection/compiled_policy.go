package detection

import (
	"strings"

	"github.com/stackrox/rox/central/searchbasedpolicies"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/compiledpolicies/deployment/predicate"
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
		pred, err := predicate.Compile(policy)
		if err != nil {
			return nil, err
		}
		compiled.predicates = append(compiled.predicates, &deploymentPredicate{
			predicate: pred,
		})
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

// AppliesTo returns if the compiled policy applies to the input object.
func (cp *compiledPolicy) AppliesTo(input interface{}) bool {
	for _, predicate := range cp.predicates {
		if predicate.AppliesTo(input) {
			return true
		}
	}
	return false
}

// Predicate says whether or not a compiled policy applies to an object.
type Predicate interface {
	AppliesTo(interface{}) bool
}

// Predicate for deployments
type deploymentPredicate struct {
	predicate predicate.Predicate
}

func (cp *deploymentPredicate) AppliesTo(input interface{}) bool {
	deployment, isDeployment := input.(*storage.Deployment)
	if !isDeployment {
		return false
	}
	return cp.predicate == nil || cp.predicate(deployment)
}

// Predicate for images.
type imagePredicate struct {
	policy *storage.Policy
}

func (cp *imagePredicate) AppliesTo(input interface{}) bool {
	image, isImage := input.(*storage.Image)
	if !isImage {
		return false
	}
	return !matchesImageWhitelist(image.GetName().GetFullName(), cp.policy.GetWhitelists())
}

func matchesImageWhitelist(image string, whitelists []*storage.Whitelist) bool {
	for _, w := range whitelists {
		if w.GetImage() == nil {
			continue
		}
		// The rationale for using a prefix is that it is the easiet way in the current format
		// to support whitelisting registries, registry/remote, etc
		if strings.HasPrefix(image, w.GetImage().GetName()) {
			return true
		}
	}
	return false
}
