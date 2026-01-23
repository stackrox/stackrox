package scopecomp

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/regexutils"
)

var (
	log = logging.LoggerForModule()
)

const (
	clusterLabelType    = "cluster label"
	namespaceLabelType  = "namespace label"
	deploymentLabelType = "deployment label"
)

// CompiledScope a transformed scope into the relevant regexes
type CompiledScope struct {
	ClusterID string

	ClusterLabelKey   regexutils.StringMatcher
	ClusterLabelValue regexutils.StringMatcher

	Namespace regexutils.StringMatcher

	NamespaceLabelKey   regexutils.StringMatcher
	NamespaceLabelValue regexutils.StringMatcher

	LabelKey   regexutils.StringMatcher
	LabelValue regexutils.StringMatcher

	clusterLabelProvider   ClusterLabelProvider
	namespaceLabelProvider NamespaceLabelProvider
}

// compileLabelMatchers compiles key and value regex matchers for a label.
func compileLabelMatchers(label *storage.Scope_Label, labelType string) (keyMatcher, valueMatcher regexutils.StringMatcher, err error) {
	if label == nil {
		return nil, nil, nil
	}

	keyMatcher, err = regexutils.CompileWholeStringMatcher(label.GetKey(), regexutils.Flags{CaseInsensitive: true})
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to compile %s key regex", labelType)
	}

	valueMatcher, err = regexutils.CompileWholeStringMatcher(label.GetValue(), regexutils.Flags{CaseInsensitive: true})
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to compile %s value regex", labelType)
	}

	return keyMatcher, valueMatcher, nil
}

// CompileScope takes in a scope and compiles it into regexes unless the regexes are invalid
func CompileScope(scope *storage.Scope, clusterLabelProvider ClusterLabelProvider, namespaceLabelProvider NamespaceLabelProvider) (*CompiledScope, error) {
	namespaceReg, err := regexutils.CompileWholeStringMatcher(scope.GetNamespace(), regexutils.Flags{CaseInsensitive: true})
	if err != nil {
		return nil, errors.Errorf("namespace regex %q could not be compiled", err)
	}

	cs := &CompiledScope{
		ClusterID:              scope.GetCluster(),
		Namespace:              namespaceReg,
		clusterLabelProvider:   clusterLabelProvider,
		namespaceLabelProvider: namespaceLabelProvider,
	}

	if features.LabelBasedPolicyScoping.Enabled() {
		cs.ClusterLabelKey, cs.ClusterLabelValue, err = compileLabelMatchers(scope.GetClusterLabel(), clusterLabelType)
		if err != nil {
			return nil, err
		}

		cs.NamespaceLabelKey, cs.NamespaceLabelValue, err = compileLabelMatchers(scope.GetNamespaceLabel(), namespaceLabelType)
		if err != nil {
			return nil, err
		}
	}

	cs.LabelKey, cs.LabelValue, err = compileLabelMatchers(scope.GetLabel(), deploymentLabelType)
	if err != nil {
		return nil, err
	}
	return cs, nil
}

// MatchesClusterLabels evaluates cluster label matchers against a deployment's cluster
func (c *CompiledScope) MatchesClusterLabels(deployment *storage.Deployment) bool {
	if !features.LabelBasedPolicyScoping.Enabled() || c.ClusterLabelKey == nil {
		return true
	}
	if c.clusterLabelProvider == nil {
		log.Error("Cluster label matcher defined but provider is nil - failing closed")
		return false
	}
	clusterLabels, err := c.clusterLabelProvider.GetClusterLabels(deployment.GetClusterId())
	if err != nil {
		log.Errorf("Failed to fetch cluster labels for cluster %s: %v", deployment.GetClusterId(), err)
		return false
	}
	return c.MatchesLabels(c.ClusterLabelKey, c.ClusterLabelValue, clusterLabels)
}

// MatchesNamespaceLabels evaluates namespace label matchers against a deployment's namespace
func (c *CompiledScope) MatchesNamespaceLabels(deployment *storage.Deployment) bool {
	if !features.LabelBasedPolicyScoping.Enabled() || c.NamespaceLabelKey == nil {
		return true
	}
	if c.namespaceLabelProvider == nil {
		log.Error("Namespace label matcher defined but provider is nil - failing closed")
		return false
	}
	namespaceLabels, err := c.namespaceLabelProvider.GetNamespaceLabels(deployment.GetNamespace())
	if err != nil {
		log.Errorf("Failed to fetch namespace labels for namespace %s: %v", deployment.GetNamespace(), err)
		return false
	}
	return c.MatchesLabels(c.NamespaceLabelKey, c.NamespaceLabelValue, namespaceLabels)
}

// MatchesDeployment evaluates a compiled scope against a deployment
func (c *CompiledScope) MatchesDeployment(deployment *storage.Deployment) bool {
	if c == nil {
		return true
	}
	if !c.MatchesCluster(deployment.GetClusterId()) {
		return false
	}
	if !c.MatchesClusterLabels(deployment) {
		return false
	}
	if !c.MatchesNamespace(deployment.GetNamespace()) {
		return false
	}
	if !c.MatchesNamespaceLabels(deployment) {
		return false
	}
	if !c.MatchesLabels(c.LabelKey, c.LabelValue, deployment.GetLabels()) {
		return false
	}
	return true
}

func (c *CompiledScope) MatchesLabels(keyMatcher regexutils.StringMatcher, valueMatcher regexutils.StringMatcher, labels map[string]string) bool {
	if keyMatcher == nil {
		return true
	}
	for key, value := range labels {
		if keyMatcher.MatchString(key) && valueMatcher.MatchString(value) {
			return true
		}
	}
	return false
}

// MatchesNamespace evaluates a compiled scope against a namespace
func (c *CompiledScope) MatchesNamespace(ns string) bool {
	if c == nil {
		return true
	}
	return c.Namespace.MatchString(ns)
}

// MatchesCluster evaluates a compiled scope against a cluster ID
func (c *CompiledScope) MatchesCluster(cluster string) bool {
	if c == nil {
		return true
	}
	return c.ClusterID == "" || c.ClusterID == cluster
}

// MatchesAuditEvent evaluates a compiled scope against a kubernetes event
func (c *CompiledScope) MatchesAuditEvent(auditEvent *storage.KubernetesEvent) bool {
	if c == nil {
		return true
	}
	if !c.MatchesCluster(auditEvent.GetObject().GetClusterId()) {
		return false
	}
	if !c.MatchesNamespace(auditEvent.GetObject().GetNamespace()) {
		return false
	}
	return true
}
