package generator

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/namespaces"
)

func createNamespacesByNameMap(namespaces []*storage.NamespaceMetadata) map[string]*storage.NamespaceMetadata {
	result := make(map[string]*storage.NamespaceMetadata, len(namespaces))

	for _, ns := range namespaces {
		result[ns.GetName()] = ns
	}
	return result
}

func labelSelectorForNamespace(ns *storage.NamespaceMetadata) *storage.LabelSelector {
	var matchLabels map[string]string

	nsLabels := ns.GetLabels()
	if nsLabels[namespaces.NamespaceNameLabel] == ns.GetName() {
		matchLabels = map[string]string{
			namespaces.NamespaceNameLabel: ns.GetName(),
		}
	} else if nsLabels[namespaces.NamespaceIDLabel] == ns.GetId() {
		matchLabels = map[string]string{
			namespaces.NamespaceIDLabel: ns.GetId(),
		}
	} else {
		matchLabels = nsLabels
	}

	return &storage.LabelSelector{
		MatchLabels: matchLabels,
	}
}
