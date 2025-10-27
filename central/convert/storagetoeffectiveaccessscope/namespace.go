package storagetoeffectiveaccessscope

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac/effectiveaccessscope"
)

func Namespaces(namespaces []*storage.NamespaceMetadata) []effectiveaccessscope.Namespace {
	if namespaces == nil {
		return nil
	}
	result := make([]effectiveaccessscope.Namespace, 0, len(namespaces))
	for _, ns := range namespaces {
		result = append(result, ns)
	}
	return result
}
