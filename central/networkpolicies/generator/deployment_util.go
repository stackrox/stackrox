package generator

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/kubernetes"
	"github.com/stackrox/rox/pkg/namespaces"
)

func isProtectedNamespace(ns string) bool {
	return ns == namespaces.StackRox || kubernetes.IsSystemNamespace(ns)
}

func isProtectedDeployment(deployment *storage.Deployment) bool {
	return isProtectedNamespace(deployment.GetNamespace())
}

func labelSelectorForDeployment(deployment *storage.Deployment) *storage.LabelSelector {
	if ls := deployment.GetLabelSelector(); ls != nil {
		return ls
	}
	return &storage.LabelSelector{
		MatchLabels: deployment.GetPodLabels(),
	}
}
