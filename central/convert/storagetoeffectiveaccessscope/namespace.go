package storagetoeffectiveaccessscope

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac/effectiveaccessscope"
)

func Namespaces(namespaces []*storage.NamespaceMetadata) []effectiveaccessscope.Namespace {
	if namespaces == nil {
		return nil
	}
	result := make([]effectiveaccessscope.Namespace, len(namespaces))
	for ix, ns := range namespaces {
		result[ix] = ns
	}
	return result
}
