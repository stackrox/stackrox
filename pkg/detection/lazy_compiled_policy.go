package detection

import (
	"context"
	"sync"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy"
	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
	"github.com/stackrox/rox/pkg/scopecomp"
)

// lazyCompiledPolicy defers the expensive compilation (regexp building,
// matcher construction) until the policy is first evaluated. This saves
// ~6 MB on idle sensors where most policies are never checked.
type lazyCompiledPolicy struct {
	policy                 *storage.Policy
	clusterLabelProvider   scopecomp.ClusterLabelProvider
	namespaceLabelProvider scopecomp.NamespaceLabelProvider

	once       sync.Once
	compiled   CompiledPolicy
	compileErr error
}

func newLazyCompiledPolicy(policy *storage.Policy, clp scopecomp.ClusterLabelProvider, nlp scopecomp.NamespaceLabelProvider) CompiledPolicy {
	return &lazyCompiledPolicy{
		policy:                 policy,
		clusterLabelProvider:   clp,
		namespaceLabelProvider: nlp,
	}
}

func (l *lazyCompiledPolicy) ensureCompiled() error {
	l.once.Do(func() {
		l.compiled, l.compileErr = newCompiledPolicy(l.policy, l.clusterLabelProvider, l.namespaceLabelProvider)
	})
	return l.compileErr
}

func (l *lazyCompiledPolicy) Policy() *storage.Policy {
	return l.policy
}

func (l *lazyCompiledPolicy) RequiresImageEnrichment() bool {
	if err := l.ensureCompiled(); err != nil {
		return false
	}
	return l.compiled.RequiresImageEnrichment()
}

func (l *lazyCompiledPolicy) AppliesTo(ctx context.Context, input interface{}) bool {
	if err := l.ensureCompiled(); err != nil {
		return false
	}
	return l.compiled.AppliesTo(ctx, input)
}

func (l *lazyCompiledPolicy) MatchAgainstDeploymentAndProcess(cache *booleanpolicy.CacheReceptacle, enhanced booleanpolicy.EnhancedDeployment, pi *storage.ProcessIndicator, processNotInBaseline bool) (booleanpolicy.Violations, error) {
	if err := l.ensureCompiled(); err != nil {
		return booleanpolicy.Violations{}, err
	}
	return l.compiled.MatchAgainstDeploymentAndProcess(cache, enhanced, pi, processNotInBaseline)
}

func (l *lazyCompiledPolicy) MatchAgainstDeployment(cache *booleanpolicy.CacheReceptacle, enhanced booleanpolicy.EnhancedDeployment) (booleanpolicy.Violations, error) {
	if err := l.ensureCompiled(); err != nil {
		return booleanpolicy.Violations{}, err
	}
	return l.compiled.MatchAgainstDeployment(cache, enhanced)
}

func (l *lazyCompiledPolicy) MatchAgainstImage(cache *booleanpolicy.CacheReceptacle, image *storage.Image) (booleanpolicy.Violations, error) {
	if err := l.ensureCompiled(); err != nil {
		return booleanpolicy.Violations{}, err
	}
	return l.compiled.MatchAgainstImage(cache, image)
}

func (l *lazyCompiledPolicy) MatchAgainstKubeResourceAndEvent(cache *booleanpolicy.CacheReceptacle, kubeEvent *storage.KubernetesEvent, kubeResource interface{}) (booleanpolicy.Violations, error) {
	if err := l.ensureCompiled(); err != nil {
		return booleanpolicy.Violations{}, err
	}
	return l.compiled.MatchAgainstKubeResourceAndEvent(cache, kubeEvent, kubeResource)
}

func (l *lazyCompiledPolicy) MatchAgainstAuditLogEvent(cache *booleanpolicy.CacheReceptacle, kubeEvent *storage.KubernetesEvent) (booleanpolicy.Violations, error) {
	if err := l.ensureCompiled(); err != nil {
		return booleanpolicy.Violations{}, err
	}
	return l.compiled.MatchAgainstAuditLogEvent(cache, kubeEvent)
}

func (l *lazyCompiledPolicy) MatchAgainstDeploymentAndNetworkFlow(cache *booleanpolicy.CacheReceptacle, enhanced booleanpolicy.EnhancedDeployment, flow *augmentedobjs.NetworkFlowDetails) (booleanpolicy.Violations, error) {
	if err := l.ensureCompiled(); err != nil {
		return booleanpolicy.Violations{}, err
	}
	return l.compiled.MatchAgainstDeploymentAndNetworkFlow(cache, enhanced, flow)
}

func (l *lazyCompiledPolicy) MatchAgainstNodeAndFileAccess(cache *booleanpolicy.CacheReceptacle, node *storage.Node, access *storage.FileAccess) (booleanpolicy.Violations, error) {
	if err := l.ensureCompiled(); err != nil {
		return booleanpolicy.Violations{}, err
	}
	return l.compiled.MatchAgainstNodeAndFileAccess(cache, node, access)
}

func (l *lazyCompiledPolicy) MatchAgainstDeploymentAndFileAccess(cache *booleanpolicy.CacheReceptacle, enhanced booleanpolicy.EnhancedDeployment, access *storage.FileAccess) (booleanpolicy.Violations, error) {
	if err := l.ensureCompiled(); err != nil {
		return booleanpolicy.Violations{}, err
	}
	return l.compiled.MatchAgainstDeploymentAndFileAccess(cache, enhanced, access)
}
