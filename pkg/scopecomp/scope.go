package scopecomp

import (
	"strings"

	"github.com/stackrox/rox/generated/api/v1"
)

// WithinScope evaluates if the deployment is within the scope
func WithinScope(scope *v1.Scope, deployment *v1.Deployment) bool {
	if cluster := scope.GetCluster(); cluster != "" && deployment.GetClusterId() != cluster {
		return false
	}

	if namespace := scope.GetNamespace(); namespace != "" && deployment.GetNamespace() != namespace {
		return false
	}

	if scope.GetLabel() == nil {
		return true
	}

	labelMap := make(map[string]string, len(deployment.GetLabels()))
	for _, label := range deployment.GetLabels() {
		labelMap[strings.ToLower(label.GetKey())] = strings.ToLower(label.GetValue())
	}

	if value, ok := labelMap[strings.ToLower(scope.GetLabel().GetKey())]; !ok || strings.ToLower(value) != scope.GetLabel().GetValue() {
		return false
	}

	return true
}
