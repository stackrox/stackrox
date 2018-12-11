package scopecomp

import (
	"strings"

	"github.com/stackrox/rox/generated/storage"
)

// WithinScope evaluates if the deployment is within the scope
func WithinScope(scope *storage.Scope, deployment *storage.Deployment) bool {
	if cluster := scope.GetCluster(); cluster != "" && deployment.GetClusterId() != cluster {
		return false
	}

	if namespace := scope.GetNamespace(); namespace != "" && deployment.GetNamespace() != namespace {
		return false
	}

	if scope.GetLabel() == nil {
		return true
	}

	if value, ok := deployment.GetLabels()[strings.ToLower(scope.GetLabel().GetKey())]; !ok || strings.ToLower(value) != scope.GetLabel().GetValue() {
		return false
	}

	return true
}
