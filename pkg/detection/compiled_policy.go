package detection

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/policies"
	"github.com/stackrox/rox/pkg/scopecomp"
	"github.com/stackrox/rox/pkg/searchbasedpolicies"
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

	whitelists := make([]*compiledWhitelist, 0, len(policy.GetWhitelists()))
	for _, w := range policy.GetWhitelists() {
		w, err := newCompiledWhitelist(w)
		if err != nil {
			return nil, err
		}
		whitelists = append(whitelists, w)
	}

	if policies.AppliesAtDeployTime(policy) || policies.AppliesAtRunTime(policy) {
		compiled.predicates = append(compiled.predicates, &deploymentPredicate{policy: policy, whitelists: whitelists})
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

type compiledWhitelist struct {
	whitelist *storage.Whitelist
	cs        *scopecomp.CompiledScope
}

func newCompiledWhitelist(whitelist *storage.Whitelist) (*compiledWhitelist, error) {
	if whitelist.GetDeployment() == nil || whitelist.GetDeployment().GetScope() == nil {
		return &compiledWhitelist{
			whitelist: whitelist,
		}, nil
	}

	cs, err := scopecomp.CompileScope(whitelist.GetDeployment().GetScope())
	if err != nil {
		return nil, err
	}
	return &compiledWhitelist{
		whitelist: whitelist,
		cs:        cs,
	}, nil
}

func (cw *compiledWhitelist) MatchesDeployment(deployment *storage.Deployment) bool {
	if whitelistIsExpired(cw.whitelist) {
		return false
	}
	deploymentWhitelist := cw.whitelist.GetDeployment()
	if deploymentWhitelist == nil {
		return false
	}

	if !cw.cs.MatchesDeployment(deployment) {
		return false
	}

	if deploymentWhitelist.GetName() != "" && deploymentWhitelist.GetName() != deployment.GetName() {
		return false
	}
	return true
}

// Predicate for deployments.
type deploymentPredicate struct {
	policy     *storage.Policy
	whitelists []*compiledWhitelist
}

func (cp *deploymentPredicate) AppliesTo(input interface{}) bool {
	deployment, isDeployment := input.(*storage.Deployment)
	if !isDeployment {
		return false
	}

	return !matchesDeploymentWhitelists(deployment, cp.whitelists)
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
	return !matchesImageWhitelist(image.GetName().GetFullName(), cp.policy)
}
