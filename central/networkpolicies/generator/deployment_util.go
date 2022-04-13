package generator

import (
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/kubernetes"
	"github.com/stackrox/stackrox/pkg/namespaces"
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
