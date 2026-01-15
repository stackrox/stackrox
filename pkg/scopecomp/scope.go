package scopecomp

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/regexutils"
)

// CompiledScope a transformed scope into the relevant regexes
type CompiledScope struct {
	ClusterID string
	Namespace regexutils.StringMatcher

	LabelKey   regexutils.StringMatcher
	LabelValue regexutils.StringMatcher

	ClusterLabelKey     regexutils.StringMatcher
	ClusterLabelValue   regexutils.StringMatcher
	NamespaceLabelKey   regexutils.StringMatcher
	NamespaceLabelValue regexutils.StringMatcher
}

// CompileScope takes in a scope and compiles it into regexes unless the regexes are invalid
func CompileScope(scope *storage.Scope) (*CompiledScope, error) {
	namespaceReg, err := regexutils.CompileWholeStringMatcher(scope.GetNamespace(), regexutils.Flags{CaseInsensitive: true})
	if err != nil {
		return nil, errors.Errorf("namespace regex %q could not be compiled", err)
	}

	cs := &CompiledScope{
		ClusterID: scope.GetCluster(),
		Namespace: namespaceReg,
	}

	if scope.GetLabel() != nil {
		cs.LabelKey, err = regexutils.CompileWholeStringMatcher(scope.GetLabel().GetKey(), regexutils.Flags{CaseInsensitive: true})
		if err != nil {
			return nil, errors.Errorf("label key regex %q could not be compiled", err)
		}
		if cs.LabelKey == nil {
			return nil, errors.Errorf("label %q=%q is invalid", scope.GetLabel().GetKey(), scope.GetLabel().GetValue())
		}

		cs.LabelValue, err = regexutils.CompileWholeStringMatcher(scope.GetLabel().GetValue(), regexutils.Flags{CaseInsensitive: true})
		if err != nil {
			return nil, errors.Errorf("label value regex %q could not be compiled", err)
		}
	}

	if scope.GetClusterLabel() != nil {
		cs.ClusterLabelKey, err = regexutils.CompileWholeStringMatcher(scope.GetClusterLabel().GetKey(), regexutils.Flags{CaseInsensitive: true})
		if err != nil {
			return nil, errors.Errorf("cluster label key regex %q could not be compiled", err)
		}
		if cs.ClusterLabelKey == nil {
			return nil, errors.Errorf("cluster label %q=%q is invalid", scope.GetClusterLabel().GetKey(), scope.GetClusterLabel().GetValue())
		}

		cs.ClusterLabelValue, err = regexutils.CompileWholeStringMatcher(scope.GetClusterLabel().GetValue(), regexutils.Flags{CaseInsensitive: true})
		if err != nil {
			return nil, errors.Errorf("cluster label value regex %q could not be compiled", err)
		}
	}

	if scope.GetNamespaceLabel() != nil {
		cs.NamespaceLabelKey, err = regexutils.CompileWholeStringMatcher(scope.GetNamespaceLabel().GetKey(), regexutils.Flags{CaseInsensitive: true})
		if err != nil {
			return nil, errors.Errorf("namespace label key regex %q could not be compiled", err)
		}
		if cs.NamespaceLabelKey == nil {
			return nil, errors.Errorf("namespace label %q=%q is invalid", scope.GetNamespaceLabel().GetKey(), scope.GetNamespaceLabel().GetValue())
		}

		cs.NamespaceLabelValue, err = regexutils.CompileWholeStringMatcher(scope.GetNamespaceLabel().GetValue(), regexutils.Flags{CaseInsensitive: true})
		if err != nil {
			return nil, errors.Errorf("namespace label value regex %q could not be compiled", err)
		}
	}

	return cs, nil
}

// MatchesDeployment evaluates a compiled scope against a deployment.
// All specified filters must match (AND logic):
// - cluster ID/labels must match if specified
// - namespace name/labels must match if specified
// - deployment labels must match if specified
func (c *CompiledScope) MatchesDeployment(deployment *storage.Deployment, clusterLabels, namespaceLabels map[string]string) bool {
	if c == nil {
		return true
	}
	if !c.MatchesCluster(deployment.GetClusterId(), clusterLabels) {
		return false
	}
	if !c.MatchesNamespace(deployment.GetNamespace(), namespaceLabels) {
		return false
	}

	if c.LabelKey == nil {
		return true
	}

	var matched bool
	for key, value := range deployment.GetLabels() {
		if c.LabelKey.MatchString(key) && c.LabelValue.MatchString(value) {
			matched = true
			break
		}
	}
	return matched
}

// MatchesNamespace evaluates a compiled scope against a namespace
func (c *CompiledScope) MatchesNamespace(ns string, namespaceLabels map[string]string) bool {
	if c == nil {
		return true
	}
	if !c.Namespace.MatchString(ns) {
		return false
	}

	// If no namespace label filter is set, match based on namespace name only
	if c.NamespaceLabelKey == nil {
		return true
	}

	// Check if any namespace label matches the filter.
	// Note: If namespaceLabels is nil (no label data available), this conservatively
	// returns false (fail-closed) - policies with label filters won't match until
	// label data is provided by callers.
	for key, value := range namespaceLabels {
		if c.NamespaceLabelKey.MatchString(key) && c.NamespaceLabelValue.MatchString(value) {
			return true
		}
	}
	return false
}

// MatchesCluster evaluates a compiled scope against a cluster ID
func (c *CompiledScope) MatchesCluster(cluster string, clusterLabels map[string]string) bool {
	if c == nil {
		return true
	}
	if c.ClusterID != "" && c.ClusterID != cluster {
		return false
	}

	// If no cluster label filter is set, match based on cluster ID only
	if c.ClusterLabelKey == nil {
		return true
	}

	// Check if any cluster label matches the filter.
	// Note: If clusterLabels is nil (no label data available), this conservatively
	// returns false (fail-closed) - policies with label filters won't match until
	// label data is provided by callers.
	for key, value := range clusterLabels {
		if c.ClusterLabelKey.MatchString(key) && c.ClusterLabelValue.MatchString(value) {
			return true
		}
	}
	return false
}

// MatchesAuditEvent evaluates a compiled scope against a kubernetes event
func (c *CompiledScope) MatchesAuditEvent(auditEvent *storage.KubernetesEvent) bool {
	if c == nil {
		return true
	}
	// Pass nil for labels since audit events don't currently support label-based matching
	if !c.MatchesCluster(auditEvent.GetObject().GetClusterId(), nil) {
		return false
	}
	if !c.MatchesNamespace(auditEvent.GetObject().GetNamespace(), nil) {
		return false
	}
	return true
}
