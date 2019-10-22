package scopecomp

import (
	"regexp"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/regexutils"
)

// CompiledScope a transformed scope into the relevant regexes
type CompiledScope struct {
	ClusterID  string
	Namespace  *regexp.Regexp
	LabelKey   *regexp.Regexp
	LabelValue *regexp.Regexp
}

// CompileScope takes in a scope and compiles it into regexes unless the regexes are invalid
func CompileScope(scope *storage.Scope) (*CompiledScope, error) {
	namespaceReg, err := regexp.Compile(scope.GetNamespace())
	if err != nil {
		return nil, errors.Errorf("namespace regex %q could not be compiled", err)
	}

	cs := &CompiledScope{
		ClusterID: scope.GetCluster(),
		Namespace: namespaceReg,
	}

	if scope.GetLabel() != nil {
		cs.LabelKey, err = regexp.Compile(scope.GetLabel().GetKey())
		if err != nil {
			return nil, errors.Errorf("label key regex %q could not be compiled", err)
		}
		cs.LabelValue, err = regexp.Compile(scope.GetLabel().GetValue())
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
	if c.ClusterID != "" && c.ClusterID != deployment.GetClusterId() {
		return false
	}
	if !regexutils.MatchWholeString(c.Namespace, deployment.GetNamespace()) {
		return false
	}

	if c.LabelKey == nil {
		return true
	}

	var matched bool
	for key, value := range deployment.GetLabels() {
		if regexutils.MatchWholeString(c.LabelKey, key) && regexutils.MatchWholeString(c.LabelValue, value) {
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
	return c.Namespace.MatchString(ns)
}

// MatchesCluster evaluates a compiled scope against a cluster ID
func (c *CompiledScope) MatchesCluster(cluster string) bool {
	return c.ClusterID == cluster
}
