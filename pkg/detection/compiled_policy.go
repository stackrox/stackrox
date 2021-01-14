package detection

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/policies"
	"github.com/stackrox/rox/pkg/scopecomp"
)

// CompiledPolicy is a compiled policy, which means it can match a policy, as well as check whether a policy is applicable.
type CompiledPolicy interface {
	Policy() *storage.Policy

	// The Match* functions return violations for the policy against the passed in objects.
	// Note that the Match* functions DO NOT care about excludes scopes, or the policy being disabled.
	// Callers are responsible for doing those checks separately.
	// For MatchAgainstDeployment* functions, images _must_ correspond one-to-one with the container specs in the deployment.
	MatchAgainstDeploymentAndProcess(cacheReceptacle *booleanpolicy.CacheReceptacle, deployment *storage.Deployment, images []*storage.Image, pi *storage.ProcessIndicator, processNotInBaseline bool) (booleanpolicy.Violations, error)
	MatchAgainstDeployment(cacheReceptacle *booleanpolicy.CacheReceptacle, deployment *storage.Deployment, images []*storage.Image) (booleanpolicy.Violations, error)
	MatchAgainstImage(cacheReceptacle *booleanpolicy.CacheReceptacle, image *storage.Image) (booleanpolicy.Violations, error)
	MatchAgainstKubeResourceAndEvent(cacheReceptacle *booleanpolicy.CacheReceptacle, kubeEvent *storage.KubernetesEvent, kubeResource interface{}) (booleanpolicy.Violations, error)

	Predicate
}

// newCompiledPolicy creates and returns a compiled policy from the policy and legacySearchBasedMatcher.
func newCompiledPolicy(policy *storage.Policy) (CompiledPolicy, error) {
	compiled := &compiledPolicy{
		policy: policy,
	}

	if policies.AppliesAtRunTime(policy) {
		// TODO: Change inverse filter to filter by process fields once section validation is added to prohibit deploy time only fields.
		filtered := booleanpolicy.FilterPolicySections(policy, func(section *storage.PolicySection) bool {
			return !booleanpolicy.SectionContainsOneOf(section, booleanpolicy.KubeEventsFields)
		})
		if len(filtered.GetPolicySections()) > 0 {
			compiled.hasProcessSection = true
			deploymentWithProcessMatcher, err := booleanpolicy.BuildDeploymentWithProcessMatcher(filtered)
			if err != nil {
				return nil, errors.Wrapf(err, "building process matcher for policy %q", policy.GetName())
			}
			compiled.deploymentWithProcessMatcher = deploymentWithProcessMatcher
		}

		// Historically deploy time only field sections in policy were allowed. For kube event policies (and eventually
		// all runtime policies), we do not want to allow such sections. If a section does not contain a kube event
		// field, it implies it does not apply to kubernetes event.
		filtered = booleanpolicy.FilterPolicySections(policy, func(section *storage.PolicySection) bool {
			return booleanpolicy.SectionContainsOneOf(section, booleanpolicy.KubeEventsFields)
		})
		if len(filtered.GetPolicySections()) > 0 {
			compiled.hasKubeEventsSection = true
			kubeEventsMatcher, err := booleanpolicy.BuildKubeEventMatcher(filtered)
			if err != nil {
				return nil, errors.Wrapf(err, "building kubernetes event matcher for policy %q", policy.GetName())
			}
			compiled.kubeEventsMatcher = kubeEventsMatcher
		}
		if features.K8sEventDetection.Enabled() {
			if compiled.deploymentWithProcessMatcher == nil && compiled.kubeEventsMatcher == nil {
				return nil, errors.Errorf("incorrect sections for a runtime policy %q. Section must have least "+
					"one runtime constraint from process or kubernetes event category, but not both", policy.GetName())
			}
		}
	}

	if policies.AppliesAtDeployTime(policy) {
		deploymentMatcher, err := booleanpolicy.BuildDeploymentMatcher(policy)
		if err != nil {
			return nil, errors.Wrapf(err, "building deployment matcher for policy %q", policy.GetName())
		}
		compiled.deploymentMatcher = deploymentMatcher
	}

	if policies.AppliesAtBuildTime(policy) {
		imageMatcher, err := booleanpolicy.BuildImageMatcher(policy)
		if err != nil {
			return nil, errors.Wrapf(err, "building image matcher for policy %q", policy.GetName())
		}
		compiled.imageMatcher = imageMatcher
	}

	if compiled.deploymentMatcher == nil && compiled.imageMatcher == nil &&
		compiled.deploymentWithProcessMatcher == nil && compiled.kubeEventsMatcher == nil {
		return nil, errors.Errorf("no known lifecycle stage in policy %q", policy.GetName())
	}

	exclusions := make([]*compiledExclusion, 0, len(policy.GetWhitelists()))
	for _, w := range policy.GetWhitelists() {
		w, err := newCompiledExclusion(w)
		if err != nil {
			return nil, err
		}
		exclusions = append(exclusions, w)
	}

	scopes := make([]*scopecomp.CompiledScope, 0, len(policy.GetScope()))
	for _, s := range policy.GetScope() {
		compiled, err := scopecomp.CompileScope(s)
		if err != nil {
			return nil, errors.Wrapf(err, "compiling scope %+v for policy %q", s, policy.GetName())
		}
		scopes = append(scopes, compiled)
	}

	if policies.AppliesAtDeployTime(policy) || policies.AppliesAtRunTime(policy) {
		compiled.predicates = append(compiled.predicates, &deploymentPredicate{scopes: scopes, exclusions: exclusions})
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

	kubeEventsMatcher            booleanpolicy.KubeEventMatcher
	deploymentWithProcessMatcher booleanpolicy.DeploymentWithProcessMatcher
	deploymentMatcher            booleanpolicy.DeploymentMatcher
	imageMatcher                 booleanpolicy.ImageMatcher

	hasProcessSection    bool
	hasKubeEventsSection bool
}

func (cp *compiledPolicy) MatchAgainstKubeResourceAndEvent(
	cache *booleanpolicy.CacheReceptacle,
	kubeEvent *storage.KubernetesEvent,
	kubeResource interface{},
) (booleanpolicy.Violations, error) {
	if !cp.hasKubeEventsSection {
		return booleanpolicy.Violations{}, nil
	}

	if cp.kubeEventsMatcher == nil {
		return booleanpolicy.Violations{}, errors.Errorf("couldn't match policy %s against kubernetes event", cp.Policy().GetName())
	}
	return cp.kubeEventsMatcher.MatchKubeEvent(cache, kubeEvent, kubeResource)
}

func (cp *compiledPolicy) MatchAgainstDeploymentAndProcess(
	cache *booleanpolicy.CacheReceptacle,
	deployment *storage.Deployment,
	images []*storage.Image,
	pi *storage.ProcessIndicator,
	processNotInBaseline bool,
) (booleanpolicy.Violations, error) {
	if !cp.hasProcessSection {
		return booleanpolicy.Violations{}, nil
	}

	if cp.deploymentWithProcessMatcher == nil {
		return booleanpolicy.Violations{}, errors.Errorf("couldn't match policy %q against deployments and processes", cp.Policy().GetName())
	}
	return cp.deploymentWithProcessMatcher.MatchDeploymentWithProcess(cache, deployment, images, pi, processNotInBaseline)
}

func (cp *compiledPolicy) MatchAgainstDeployment(cache *booleanpolicy.CacheReceptacle, deployment *storage.Deployment, images []*storage.Image) (booleanpolicy.Violations, error) {
	if cp.deploymentMatcher == nil {
		return booleanpolicy.Violations{}, errors.Errorf("couldn't match policy %q against deployments", cp.Policy().GetName())
	}
	return cp.deploymentMatcher.MatchDeployment(cache, deployment, images)
}

func (cp *compiledPolicy) MatchAgainstImage(cache *booleanpolicy.CacheReceptacle, image *storage.Image) (booleanpolicy.Violations, error) {
	if cp.imageMatcher == nil {
		return booleanpolicy.Violations{}, errors.Errorf("couldn't match policy %q against images", cp.Policy().GetName())
	}
	return cp.imageMatcher.MatchImage(cache, image)
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

type compiledExclusion struct {
	exclusion *storage.Exclusion
	cs        *scopecomp.CompiledScope
}

func newCompiledExclusion(exclusion *storage.Exclusion) (*compiledExclusion, error) {
	if exclusion.GetDeployment() == nil || exclusion.GetDeployment().GetScope() == nil {
		return &compiledExclusion{
			exclusion: exclusion,
		}, nil
	}

	cs, err := scopecomp.CompileScope(exclusion.GetDeployment().GetScope())
	if err != nil {
		return nil, err
	}
	return &compiledExclusion{
		exclusion: exclusion,
		cs:        cs,
	}, nil
}

func (cw *compiledExclusion) MatchesDeployment(deployment *storage.Deployment) bool {
	if exclusionIsExpired(cw.exclusion) {
		return false
	}
	deploymentExclusion := cw.exclusion.GetDeployment()
	if deploymentExclusion == nil {
		return false
	}

	if !cw.cs.MatchesDeployment(deployment) {
		return false
	}

	if deploymentExclusion.GetName() != "" && deploymentExclusion.GetName() != deployment.GetName() {
		return false
	}
	return true
}

// Predicate for deployments.
type deploymentPredicate struct {
	exclusions []*compiledExclusion
	scopes     []*scopecomp.CompiledScope
}

func (cp *deploymentPredicate) AppliesTo(input interface{}) bool {
	deployment, isDeployment := input.(*storage.Deployment)
	if !isDeployment {
		return false
	}

	return deploymentMatchesScopes(deployment, cp.scopes) && !deploymentMatchesExclusions(deployment, cp.exclusions)
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
	return !matchesImageExclusion(image.GetName().GetFullName(), cp.policy)
}
