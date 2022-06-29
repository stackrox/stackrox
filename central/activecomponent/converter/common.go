package converter

import (
	"sort"

	"github.com/stackrox/rox/generated/storage"
)

// ConvertActiveContextsMapToSlice creates a slice of contexts from a map
func ConvertActiveContextsMapToSlice(contextMap map[string]*storage.ActiveComponent_ActiveContext) []*storage.ActiveComponent_ActiveContext {
	contexts := make([]*storage.ActiveComponent_ActiveContext, 0, len(contextMap))
	for _, ctx := range contextMap {
		contexts = append(contexts, ctx)
	}
	sort.SliceStable(contexts, func(i, j int) bool {
		return contexts[i].ContainerName < contexts[j].ContainerName
	})
	return contexts
}
