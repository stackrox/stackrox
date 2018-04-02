package scopecomp

import "bitbucket.org/stack-rox/apollo/generated/api/v1"

// WithinScope evaluates if the deployment is within the scope
func WithinScope(scope *v1.Scope, deployment *v1.Deployment) bool {
	if cluster := scope.GetCluster(); cluster != "" && deployment.GetClusterId() != cluster {
		return false
	}

	if namespace := scope.GetNamespace(); namespace != "" && deployment.GetNamespace() != namespace {
		return false
	}

	if label := scope.GetLabel(); label != nil && deployment.GetLabels()[label.GetKey()] != label.GetValue() {
		return false
	}

	return true
}
