package generator

import "github.com/stackrox/rox/generated/storage"

func isSystemDeployment(deployment *storage.Deployment) bool {
	return deployment.GetNamespace() == `kube-system` || deployment.GetNamespace() == `kube-public`
}

func labelSelectorForDeployment(deployment *storage.Deployment) *storage.LabelSelector {
	if ls := deployment.GetLabelSelector(); ls != nil {
		return ls
	}
	return &storage.LabelSelector{
		MatchLabels: deployment.GetPodLabels(),
	}
}
