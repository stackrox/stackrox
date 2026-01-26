package scopecomp

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/regexutils"
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
	return cs, nil
}

// MatchesDeployment evaluates a compiled scope against a deployment
func (c *CompiledScope) MatchesDeployment(deployment *storage.Deployment, clusterLabels map[string]string, namespaceLabels map[string]string) bool {
	if c == nil {
		return true
	}
	if !c.MatchesCluster(deployment.GetClusterId()) {
		return false
	}
	if features.LabelBasedPolicyScoping.Enabled() {
		if !c.MatchesLabels(c.ClusterLabelKey, c.ClusterLabelValue, clusterLabels) {
			return false
		}
	}
	if !c.MatchesNamespace(deployment.GetNamespace()) {
		return false
	}
	if features.LabelBasedPolicyScoping.Enabled() {
		if !c.MatchesLabels(c.NamespaceLabelKey, c.NamespaceLabelValue, namespaceLabels) {
			return false
		}
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
