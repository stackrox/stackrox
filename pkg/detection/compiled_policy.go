package detection

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/policies"
	"github.com/stackrox/rox/pkg/scopecomp"
	"github.com/stackrox/rox/pkg/searchbasedpolicies"
)

// CompiledPolicy is a compiled policy, which means it can match a policy, as well as check whether a policy is applicable.
type CompiledPolicy interface {
	Policy() *storage.Policy

	// The Match* functions return violations for the policy against the passed in objects.
	// Note that the Match* functions DO NOT care about whitelists/the policy being disabled.
	// Callers are responsible for doing those checks separately.
	// For MatchAgainstDeployment* functions, images _must_ correspond one-to-one with the container specs in the deployment.
	MatchAgainstDeploymentAndProcess(deployment *storage.Deployment, images []*storage.Image, pi *storage.ProcessIndicator, processOutsideWhitelist bool) (searchbasedpolicies.Violations, error)
	MatchAgainstDeployment(deployment *storage.Deployment, images []*storage.Image) (searchbasedpolicies.Violations, error)
	MatchAgainstImage(image *storage.Image) (searchbasedpolicies.Violations, error)

	Predicate
}

// newCompiledPolicy creates and returns a compiled policy from the policy and legacySearchBasedMatcher.
func newCompiledPolicy(policy *storage.Policy, matcher searchbasedpolicies.Matcher) (CompiledPolicy, error) {
	compiled := &compiledPolicy{
		policy:                   policy,
		legacySearchBasedMatcher: matcher,
	}

	if features.BooleanPolicyLogic.Enabled() {
		if policies.AppliesAtDeployTime(policy) || policies.AppliesAtRunTime(policy) {
			deploymentMatcher, err := booleanpolicy.BuildDeploymentMatcher(policy)
			if err != nil {
				return nil, errors.Wrap(err, "building deployment matcher")
			}
			compiled.deploymentMatcher = deploymentMatcher
		}

		if policies.AppliesAtBuildTime(policy) {
			imageMatcher, err := booleanpolicy.BuildImageMatcher(policy)
			if err != nil {
				return nil, errors.Wrap(err, "building image matcher")
			}
			compiled.imageMatcher = imageMatcher
		}
	}

	whitelists := make([]*compiledWhitelist, 0, len(policy.GetWhitelists()))
	for _, w := range policy.GetWhitelists() {
		w, err := newCompiledWhitelist(w)
		if err != nil {
			return nil, err
		}
		whitelists = append(whitelists, w)
	}

	scopes := make([]*scopecomp.CompiledScope, 0, len(policy.GetScope()))
	for _, s := range policy.GetScope() {
		compiled, err := scopecomp.CompileScope(s)
		if err != nil {
			return nil, errors.Wrapf(err, "compiling scope %+v", s)
		}
		scopes = append(scopes, compiled)
	}

	if policies.AppliesAtDeployTime(policy) || policies.AppliesAtRunTime(policy) {
		compiled.predicates = append(compiled.predicates, &deploymentPredicate{scopes: scopes, whitelists: whitelists})
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
	predicates []Predicate

	legacySearchBasedMatcher searchbasedpolicies.Matcher

	deploymentMatcher booleanpolicy.DeploymentMatcher
	imageMatcher      booleanpolicy.ImageMatcher
}

func (cp *compiledPolicy) MatchAgainstDeploymentAndProcess(deployment *storage.Deployment, images []*storage.Image, pi *storage.ProcessIndicator, processOutsideWhitelist bool) (searchbasedpolicies.Violations, error) {
	if features.BooleanPolicyLogic.Enabled() {
		if cp.deploymentMatcher == nil {
			return searchbasedpolicies.Violations{}, errors.Errorf("couldn't match policy %s against deployments and processes", cp.Policy().GetName())
		}
		return cp.deploymentMatcher.MatchDeploymentWithProcess(context.Background(), deployment, images, pi, processOutsideWhitelist)
	}
	return cp.legacySearchBasedMatcher.MatchOne(context.Background(), deployment, images, pi)
}

func (cp *compiledPolicy) MatchAgainstDeployment(deployment *storage.Deployment, images []*storage.Image) (searchbasedpolicies.Violations, error) {
	if features.BooleanPolicyLogic.Enabled() {
		if cp.deploymentMatcher == nil {
			return searchbasedpolicies.Violations{}, errors.Errorf("couldn't match policy %s against deployments", cp.Policy().GetName())
		}
		return cp.deploymentMatcher.MatchDeployment(context.Background(), deployment, images)
	}
	return cp.legacySearchBasedMatcher.MatchOne(context.Background(), deployment, images, nil)
}

func (cp *compiledPolicy) MatchAgainstImage(image *storage.Image) (searchbasedpolicies.Violations, error) {
	if features.BooleanPolicyLogic.Enabled() {
		if cp.imageMatcher == nil {
			return searchbasedpolicies.Violations{}, errors.Errorf("couldn't match policy %s against images", cp.Policy().GetName())
		}
		return cp.imageMatcher.MatchImage(context.Background(), image)
	}

	return cp.legacySearchBasedMatcher.MatchOne(context.Background(), nil, []*storage.Image{image}, nil)
}

// Policy returns the policy that was compiled.
func (cp *compiledPolicy) Policy() *storage.Policy {
	return cp.policy
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
	whitelists []*compiledWhitelist
	scopes     []*scopecomp.CompiledScope
}

func (cp *deploymentPredicate) AppliesTo(input interface{}) bool {
	deployment, isDeployment := input.(*storage.Deployment)
	if !isDeployment {
		return false
	}

	// In the pre-BPL world, scopes are handled by constructing a conjunction query in the matcher.
	if !features.BooleanPolicyLogic.Enabled() {
		return !deploymentMatchesWhitelists(deployment, cp.whitelists)
	}

	return deploymentMatchesScopes(deployment, cp.scopes) && !deploymentMatchesWhitelists(deployment, cp.whitelists)
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
