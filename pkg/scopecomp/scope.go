package scopecomp

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/regexutils"
	"github.com/stackrox/rox/pkg/utils"
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

	if !features.LabelBasedPolicyScoping.Enabled() && (scope.GetClusterLabel() != nil || scope.GetNamespaceLabel() != nil) {
		return nil, errors.New("cluster_label and namespace_label scopes require ROX_LABEL_BASED_POLICY_SCOPING feature flag to be enabled")
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

// MatchesClusterLabels evaluates cluster label matchers against a cluster's labels
func (c *CompiledScope) MatchesClusterLabels(ctx context.Context, clusterID string) bool {
	if c.ClusterLabelKey == nil {
		return true
	}
	// Unreachable: all CompileScope callers for inclusion scopes pass real
	// providers, and exclusion scopes reject labels at validation time.
	if c.clusterLabelProvider == nil {
		utils.Should(errors.New("cluster label matcher defined but provider is nil"))
		return false
	}
	clusterLabels, err := c.clusterLabelProvider.GetClusterLabels(ctx, clusterID)
	if err != nil {
		log.Errorf("Failed to fetch cluster labels for cluster %s: %v", clusterID, err)
		return false
	}
	return c.MatchesLabels(c.ClusterLabelKey, c.ClusterLabelValue, clusterLabels)
}

// MatchesNamespaceLabels evaluates namespace label matchers against a namespace's labels
func (c *CompiledScope) MatchesNamespaceLabels(ctx context.Context, clusterID string, namespace string) bool {
	if c.NamespaceLabelKey == nil {
		return true
	}
	// Unreachable: all CompileScope callers for inclusion scopes pass real
	// providers, and exclusion scopes reject labels at validation time.
	if c.namespaceLabelProvider == nil {
		utils.Should(errors.New("namespace label matcher defined but provider is nil"))
		return false
	}
	namespaceLabels, err := c.namespaceLabelProvider.GetNamespaceLabels(ctx, clusterID, namespace)
	if err != nil {
		log.Errorf("Failed to fetch namespace labels for namespace %s in cluster %s: %v", namespace, clusterID, err)
		return false
	}
	return c.MatchesLabels(c.NamespaceLabelKey, c.NamespaceLabelValue, namespaceLabels)
}

// MatchesDeployment evaluates a compiled scope against a deployment
func (c *CompiledScope) MatchesDeployment(ctx context.Context, deployment *storage.Deployment) bool {
	if c == nil {
		return true
	}
	if !c.MatchesCluster(deployment.GetClusterId()) {
		return false
	}
	if !c.MatchesClusterLabels(ctx, deployment.GetClusterId()) {
		return false
	}
	if !c.MatchesNamespace(deployment.GetNamespace()) {
		return false
	}
	if !c.MatchesNamespaceLabels(ctx, deployment.GetClusterId(), deployment.GetNamespace()) {
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
func (c *CompiledScope) MatchesAuditEvent(ctx context.Context, auditEvent *storage.KubernetesEvent) bool {
	if c == nil {
		return true
	}
	if !c.MatchesCluster(auditEvent.GetObject().GetClusterId()) {
		return false
	}
	if !c.MatchesClusterLabels(ctx, auditEvent.GetObject().GetClusterId()) {
		return false
	}
	// Namespace matching is only applied for namespace-scoped resources.
	// Cluster-scoped resources (e.g. clusterroles) have an empty namespace
	// and should not be filtered out by namespace-based matchers.
	if ns := auditEvent.GetObject().GetNamespace(); ns != "" {
		if !c.MatchesNamespace(ns) {
			return false
		}
		if !c.MatchesNamespaceLabels(ctx, auditEvent.GetObject().GetClusterId(), ns) {
			return false
		}
	}
	return true
}
