package generator

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/namespaces"
)

func labelSelectorForNamespace(namespaceName string) *storage.LabelSelector {
	return &storage.LabelSelector{
		MatchLabels: map[string]string{
			namespaces.NamespaceNameLabel: namespaceName,
		},
	}
}
