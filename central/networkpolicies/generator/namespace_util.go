package generator

import (
	"maps"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/namespaces"
)

var allowAllNamespaces = &storage.LabelSelector{}

func createNamespacesByNameMap(namespaces []storage.ImmutableNamespaceMetadata) map[string]storage.ImmutableNamespaceMetadata {
	result := make(map[string]storage.ImmutableNamespaceMetadata, len(namespaces))

	for _, ns := range namespaces {
		result[ns.GetName()] = ns
	}
	return result
}

func labelSelectorForNamespace(ns storage.ImmutableNamespaceMetadata) *storage.LabelSelector {
	if ns == nil {
		return allowAllNamespaces
	}

	var matchLabels map[string]string

	nsLabels := maps.Collect(ns.GetImmutableLabels())
	labelKey := namespaces.GetFirstValidNamespaceNameLabelKey(nsLabels, ns.GetName())
	if labelKey != "" {
		matchLabels = map[string]string{
			labelKey: ns.GetName(),
		}
	} else {
		matchLabels = nsLabels
	}

	return &storage.LabelSelector{
		MatchLabels: matchLabels,
	}
}
