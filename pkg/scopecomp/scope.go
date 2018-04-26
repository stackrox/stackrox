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

	labelMap := make(map[string]string, len(deployment.GetLabels()))
	for _, label := range deployment.GetLabels() {
		labelMap[label.GetKey()] = label.GetValue()
	}
	if scope.GetLabel() != nil {
		for _, deploymentLabel := range deployment.GetLabels() {
			if deploymentLabel.GetKey() == scope.GetLabel().GetKey() && deploymentLabel.GetValue() != scope.GetLabel().GetValue() {
				return false
			}
		}
	}

	return true
}
