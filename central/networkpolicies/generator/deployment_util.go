package generator

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/kubernetes"
)

func isSystemDeployment(deployment *storage.Deployment) bool {
	return kubernetes.SystemNamespaceSet.Contains(deployment.GetNamespace())
}

func labelSelectorForDeployment(deployment *storage.Deployment) *storage.LabelSelector {
	if ls := deployment.GetLabelSelector(); ls != nil {
		return ls
	}
	return &storage.LabelSelector{
		MatchLabels: deployment.GetPodLabels(),
	}
}
