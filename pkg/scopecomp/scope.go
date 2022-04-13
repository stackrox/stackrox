package scopecomp

import (
	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/regexutils"
)

// CompiledScope a transformed scope into the relevant regexes
type CompiledScope struct {
	ClusterID string
	Namespace regexutils.WholeStringMatcher

	LabelKey   regexutils.WholeStringMatcher
	LabelValue regexutils.WholeStringMatcher
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
	return cs, nil
}

// MatchesDeployment evaluates a compiled scope against a deployment
func (c *CompiledScope) MatchesDeployment(deployment *storage.Deployment) bool {
	if c == nil {
		return true
	}
	if !c.MatchesCluster(deployment.GetClusterId()) {
		return false
	}
	if !c.MatchesNamespace(deployment.GetNamespace()) {
		return false
	}

	if c.LabelKey == nil {
		return true
	}

	var matched bool
	for key, value := range deployment.GetLabels() {
		if c.LabelKey.MatchWholeString(key) && c.LabelValue.MatchWholeString(value) {
			matched = true
			break
		}
	}
	return matched
}

// MatchesNamespace evaluates a compiled scope against a namespace
func (c *CompiledScope) MatchesNamespace(ns string) bool {
	if c == nil {
		return true
	}
	return c.Namespace.MatchWholeString(ns)
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
