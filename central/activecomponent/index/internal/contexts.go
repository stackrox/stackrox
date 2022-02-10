package internal

import "github.com/stackrox/rox/generated/storage"

// IndexedContexts contains the collection of active contexts to be indexed.
type IndexedContexts struct {
	ActiveContexts []*storage.ActiveComponent_ActiveContext
}

// ConvertToIndexContexts convert an active component to IndexedContexts for indexing.
func ConvertToIndexContexts(activeComponent *storage.ActiveComponent) *IndexedContexts {
	if activeComponent == nil || activeComponent.ActiveContexts == nil {
		return nil
	}
	ic := IndexedContexts{ActiveContexts: make([]*storage.ActiveComponent_ActiveContext, 0, len(activeComponent.GetActiveContexts()))}
	for _, v := range activeComponent.GetActiveContexts() {
		ic.ActiveContexts = append(ic.ActiveContexts, v)
	}
	return &ic
}
