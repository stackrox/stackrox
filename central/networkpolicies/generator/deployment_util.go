package generator

import "github.com/stackrox/rox/generated/storage"

func isSystemDeployment(deployment *storage.Deployment) bool {
	return deployment.GetNamespace() == `kube-system`
}
