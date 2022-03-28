package detection

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy"
	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
	"github.com/stackrox/rox/pkg/policies"
	"github.com/stackrox/rox/pkg/regexutils"
	"github.com/stackrox/rox/pkg/scopecomp"
)

// CompiledPolicy is a compiled policy, which means it can match a policy, as well as check whether a policy is applicable.
type CompiledPolicy interface {
	Policy() *storage.Policy

	// The Match* functions return violations for the policy against the passed in objects.
	// Note that the Match* functions DO NOT care about excludes scopes, or the policy being disabled.
	// Callers are responsible for doing those checks separately.
	// For MatchAgainstDeployment* functions, images _must_ correspond one-to-one with the container specs in the deployment.
	MatchAgainstDeploymentAndProcess(cacheReceptacle *booleanpolicy.CacheReceptacle, enhanced booleanpolicy.EnhancedDeployment, pi *storage.ProcessIndicator, processNotInBaseline bool) (booleanpolicy.Violations, error)
	MatchAgainstDeployment(cacheReceptacle *booleanpolicy.CacheReceptacle, enhanced booleanpolicy.EnhancedDeployment) (booleanpolicy.Violations, error)
	MatchAgainstImage(cacheReceptacle *booleanpolicy.CacheReceptacle, image *storage.Image) (booleanpolicy.Violations, error)
	MatchAgainstKubeResourceAndEvent(cacheReceptacle *booleanpolicy.CacheReceptacle, kubeEvent *storage.KubernetesEvent, kubeResource interface{}) (booleanpolicy.Violations, error)
	MatchAgainstAuditLogEvent(cacheReceptacle *booleanpolicy.CacheReceptacle, kubeEvent *storage.KubernetesEvent) (booleanpolicy.Violations, error)
	MatchAgainstDeploymentAndNetworkFlow(cacheReceptable *booleanpolicy.CacheReceptacle, enhancedDeployment booleanpolicy.EnhancedDeployment, flow *augmentedobjs.NetworkFlowDetails) (booleanpolicy.Violations, error)

	Predicate
}

// newCompiledPolicy creates and returns a compiled policy from the policy and legacySearchBasedMatcher.
func newCompiledPolicy(policy *storage.Policy) (CompiledPolicy, error) {
	compiled := &compiledPolicy{
		policy: policy,
	}

	exclusions := make([]*compiledExclusion, 0, len(policy.GetExclusions()))
	for _, w := range policy.GetExclusions() {
		w, err := newCompiledExclusion(w)
		if err != nil {
			return nil, errors.Wrapf(err, "compiling exclusion list %+v for policy %s", w, policy.GetName())
		}
		exclusions = append(exclusions, w)
	}

	scopes := make([]*scopecomp.CompiledScope, 0, len(policy.GetScope()))
	for _, s := range policy.GetScope() {
		compiledScope, err := scopecomp.CompileScope(s)
		if err != nil {
			return nil, errors.Wrapf(err, "compiling scope %+v for policy %q", s, policy.GetName())
		}
		scopes = append(scopes, compiledScope)
	}

	if policies.AppliesAtRunTime(policy) {
		if err := compiled.setRuntimeMatchers(policy); err != nil {
			return nil, err
		}
		// There should be exactly one defined
		if !compiled.exactlyOneRuntimeMatcherDefined() {
			return nil, errors.Errorf("incorrect sections for a runtime policy %q. Section must have exactly "+
				"one runtime constraint from either process, or kubernetes event category, or network baseline.", policy.GetName())
		}
		// set predicates
		compiled.predicates = append(compiled.predicates, &deploymentPredicate{scopes: scopes, exclusions: exclusions})
		if policy.GetEventSource() == storage.EventSource_AUDIT_LOG_EVENT {
			compiled.predicates = append(compiled.predicates, &auditEventPredicate{scopes: scopes, exclusions: exclusions})
		}
	}

	if policies.AppliesAtDeployTime(policy) {
		if err := compiled.setDeployTimeMatchers(policy); err != nil {
			return nil, errors.Wrapf(err, "building deployment matcher for policy %q", policy.GetName())
		}
		// set predicates
		compiled.predicates = append(compiled.predicates, &deploymentPredicate{scopes: scopes, exclusions: exclusions})
	}

	if policies.AppliesAtBuildTime(policy) {
		if err := compiled.setBuildTimeMatchers(policy); err != nil {
			return nil, errors.Wrapf(err, "building image matcher for policy %q", policy.GetName())
		}
		compiled.predicates = append(compiled.predicates, &imagePredicate{
			policy: policy,
		})
	}

	if compiled.noMatchersSet() {
		return nil, errors.Errorf("no valid policy criteria fields in policy %q, unable to set matchers", policy.GetName())
	}
	return compiled, nil
}

func (cp *compiledPolicy) noMatchersSet() bool {
	return cp.auditLogEventMatcher == nil &&
		cp.deploymentMatcher == nil &&
		cp.imageMatcher == nil &&
		cp.deploymentWithProcessMatcher == nil &&
		cp.kubeEventsMatcher == nil &&
		cp.deploymentWithNetworkFlowMatcher == nil
}

func (cp *compiledPolicy) setBuildTimeMatchers(policy *storage.Policy) error {
	imageMatcher, err := booleanpolicy.BuildImageMatcher(policy)
	if err != nil {
		return err
	}
	cp.imageMatcher = imageMatcher
	return nil
}

func (cp *compiledPolicy) setDeployTimeMatchers(policy *storage.Policy) error {
	deploymentMatcher, err := booleanpolicy.BuildDeploymentMatcher(policy)
	if err != nil {
		return err
	}
	cp.deploymentMatcher = deploymentMatcher
	return nil
}

func (cp *compiledPolicy) setRuntimeMatchers(policy *storage.Policy) error {
	if policy.GetEventSource() == storage.EventSource_AUDIT_LOG_EVENT {
		err := cp.setAuditLogEventMatcher(policy)
		if err != nil {
			return errors.Wrapf(err, "building audit log event matcher for policy %q", policy.GetName())
		}
		return nil
	}

	if policy.GetEventSource() == storage.EventSource_DEPLOYMENT_EVENT {
		err := cp.setProcessEventMatcher(policy)
		if err != nil {
			return errors.Wrapf(err, "building process event matcher for policy %q", policy.GetName())
		}
		err = cp.setKubeEventEventMatcher(policy)
		if err != nil {
			return errors.Wrapf(err, "building kube event matcher for policy %q", policy.GetName())
		}
		err = cp.setNetworkFlowEventMatcher(policy)
		if err != nil {
			return errors.Wrapf(err, "building network baseline matcher for policy %q", policy.GetName())
		}
	}
	return nil
}

func (cp *compiledPolicy) setAuditLogEventMatcher(policy *storage.Policy) error {
	filtered := booleanpolicy.FilterPolicySections(policy, func(section *storage.PolicySection) bool {
		return booleanpolicy.SectionContainsFieldOfType(section, booleanpolicy.AuditLogEvent)
	})
	if len(filtered.GetPolicySections()) > 0 {
		cp.hasAuditEventsSection = true
		auditLogEventMatcher, err := booleanpolicy.BuildAuditLogEventMatcher(filtered)
		if err != nil {
			return err
		}
		cp.auditLogEventMatcher = auditLogEventMatcher
	}
	return nil
}

func (cp *compiledPolicy) setProcessEventMatcher(policy *storage.Policy) error {
	filtered := booleanpolicy.FilterPolicySections(policy, func(section *storage.PolicySection) bool {
		return booleanpolicy.SectionContainsFieldOfType(section, booleanpolicy.Process)
	})
	if len(filtered.GetPolicySections()) > 0 {
		cp.hasProcessSection = true
		deploymentWithProcessMatcher, err := booleanpolicy.BuildDeploymentWithProcessMatcher(filtered)
		if err != nil {
			return err
		}
		cp.deploymentWithProcessMatcher = deploymentWithProcessMatcher
	}
	return nil
}

func (cp *compiledPolicy) setKubeEventEventMatcher(policy *storage.Policy) error {
	filtered := booleanpolicy.FilterPolicySections(policy, func(section *storage.PolicySection) bool {
		return booleanpolicy.SectionContainsFieldOfType(section, booleanpolicy.KubeEvent)
	})
	if len(filtered.GetPolicySections()) > 0 {
		cp.hasKubeEventsSection = true
		kubeEventsMatcher, err := booleanpolicy.BuildKubeEventMatcher(filtered)
		if err != nil {
			return err
		}
		cp.kubeEventsMatcher = kubeEventsMatcher
	}
	return nil
}

func (cp *compiledPolicy) setNetworkFlowEventMatcher(policy *storage.Policy) error {
	filtered := booleanpolicy.FilterPolicySections(policy, func(section *storage.PolicySection) bool {
		return booleanpolicy.SectionContainsFieldOfType(section, booleanpolicy.NetworkFlow)
	})
	if len(filtered.GetPolicySections()) > 0 {
		cp.hasNetworkFlowSection = true
		deploymentWithNetworkFlowMatcher, err := booleanpolicy.BuildDeploymentWithNetworkFlowMatcher(filtered)
		if err != nil {
			return err
		}
		cp.deploymentWithNetworkFlowMatcher = deploymentWithNetworkFlowMatcher
	}
	return nil
}

func (cp *compiledPolicy) exactlyOneRuntimeMatcherDefined() bool {
	var numMatchers int
	if cp.deploymentWithProcessMatcher != nil {
		numMatchers++
	}
	if cp.kubeEventsMatcher != nil {
		numMatchers++
	}
	if cp.deploymentWithNetworkFlowMatcher != nil {
		numMatchers++
	}
	if cp.auditLogEventMatcher != nil {
		numMatchers++
	}

	return numMatchers == 1
}

// Top level compiled Policy.
type compiledPolicy struct {
	policy     *storage.Policy
	predicates []Predicate

	kubeEventsMatcher                booleanpolicy.KubeEventMatcher
	deploymentWithProcessMatcher     booleanpolicy.DeploymentWithProcessMatcher
	deploymentWithNetworkFlowMatcher booleanpolicy.DeploymentWithNetworkFlowMatcher
	deploymentMatcher                booleanpolicy.DeploymentMatcher
	imageMatcher                     booleanpolicy.ImageMatcher
	auditLogEventMatcher             booleanpolicy.AuditLogEventMatcher

	hasProcessSection     bool
	hasKubeEventsSection  bool
	hasNetworkFlowSection bool
	hasAuditEventsSection bool
}

func (cp *compiledPolicy) MatchAgainstAuditLogEvent(
	cache *booleanpolicy.CacheReceptacle,
	kubeEvent *storage.KubernetesEvent,
) (booleanpolicy.Violations, error) {
	if !cp.hasAuditEventsSection {
		return booleanpolicy.Violations{}, nil
	}
	if cp.auditLogEventMatcher == nil {
		return booleanpolicy.Violations{}, errors.Errorf("couldn't match policy %s against audit log event", cp.Policy().GetName())
	}
	return cp.auditLogEventMatcher.MatchAuditLogEvent(cache, kubeEvent)
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
	enhancedDeployment booleanpolicy.EnhancedDeployment,
	pi *storage.ProcessIndicator,
	processNotInBaseline bool,
) (booleanpolicy.Violations, error) {
	if !cp.hasProcessSection {
		return booleanpolicy.Violations{}, nil
	}
	if cp.deploymentWithProcessMatcher == nil {
		return booleanpolicy.Violations{}, errors.Errorf("couldn't match policy %q against deployments and processes", cp.Policy().GetName())
	}

	return cp.deploymentWithProcessMatcher.MatchDeploymentWithProcess(cache, enhancedDeployment, pi, processNotInBaseline)
}

func (cp *compiledPolicy) MatchAgainstDeploymentAndNetworkFlow(
	cache *booleanpolicy.CacheReceptacle,
	enhancedDeployment booleanpolicy.EnhancedDeployment,
	flow *augmentedobjs.NetworkFlowDetails,
) (booleanpolicy.Violations, error) {
	if !cp.hasNetworkFlowSection {
		return booleanpolicy.Violations{}, nil
	}
	if cp.deploymentWithNetworkFlowMatcher == nil {
		return booleanpolicy.Violations{}, errors.Errorf("couldn't match policy %s against network baseline", cp.Policy().GetName())
	}
	return cp.deploymentWithNetworkFlowMatcher.MatchDeploymentWithNetworkFlowInfo(cache, enhancedDeployment, flow)
}

func (cp *compiledPolicy) MatchAgainstDeployment(cache *booleanpolicy.CacheReceptacle, enhancedDeployment booleanpolicy.EnhancedDeployment) (booleanpolicy.Violations, error) {
	if cp.deploymentMatcher == nil {
		return booleanpolicy.Violations{}, errors.Errorf("couldn't match policy %q against deployments", cp.Policy().GetName())
	}
	return cp.deploymentMatcher.MatchDeployment(cache, enhancedDeployment)
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
	exclusion             *storage.Exclusion
	deploymentNameMatcher regexutils.WholeStringMatcher
	cs                    *scopecomp.CompiledScope
}

type alwaysFalseMatcher struct{}

func (a *alwaysFalseMatcher) MatchWholeString(_ string) bool {
	return false
}

func newCompiledExclusion(exclusion *storage.Exclusion) (*compiledExclusion, error) {
	cx := &compiledExclusion{
		exclusion: exclusion,
	}
	if name := exclusion.GetDeployment().GetName(); name != "" {
		deploymentNameMatcher, err := regexutils.CompileWholeStringMatcher(name, regexutils.Flags{CaseInsensitive: true})
		if err != nil {
			// This maintains backward compatibility because, in the past, the exclusion was interpreted as an equality match.
			// We don't want to return an error because, if someone has a policy with an invalid regex here for whatever reason,
			// we don't want their central to crash on an upgrade.
			// NOTE: If it's not a valid regex, it's not going to be a valid deployment, so we don't need to actually
			// check this exclusion. So we use an alwaysFalseMatcher.
			log.Errorf("Invalid regex for deployment name exclusion %q: %v", name, err)
			cx.deploymentNameMatcher = &alwaysFalseMatcher{}
		} else {
			cx.deploymentNameMatcher = deploymentNameMatcher
		}
	}
	if scope := exclusion.GetDeployment().GetScope(); scope != nil {
		cs, err := scopecomp.CompileScope(exclusion.GetDeployment().GetScope())
		if err != nil {
			return nil, err
		}
		cx.cs = cs
	}

	return cx, nil
}

func (cw *compiledExclusion) MatchesDeployment(deployment *storage.Deployment) bool {
	if exclusionIsExpired(cw.exclusion) {
		return false
	}

	if cw.deploymentNameMatcher != nil && !cw.deploymentNameMatcher.MatchWholeString(deployment.GetName()) {
		return false
	}

	return cw.cs.MatchesDeployment(deployment)
}

func (cw *compiledExclusion) MatchesAuditEvent(auditEvent *storage.KubernetesEvent) bool {
	if exclusionIsExpired(cw.exclusion) {
		return false
	}
	if !cw.cs.MatchesAuditEvent(auditEvent) {
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

// Predicate for audit events.
type auditEventPredicate struct {
	exclusions []*compiledExclusion
	scopes     []*scopecomp.CompiledScope
}

func (cp *auditEventPredicate) AppliesTo(input interface{}) bool {
	auditEvent, isAuditEvent := input.(*storage.KubernetesEvent)
	if !isAuditEvent {
		return false
	}

	return auditEventMatchesScopes(auditEvent, cp.scopes) && !auditEventMatchesExclusions(auditEvent, cp.exclusions)
}
