package resources

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	v1 "k8s.io/api/core/v1"
)

// NamespaceDeletionListener allows components to react to the deletion of namespaces.
type NamespaceDeletionListener interface {
	OnNamespaceDeleted(string)
}

// namespaceDispatcher handles namespace resource events.
type namespaceDispatcher struct {
	nsStore           *namespaceStore
	deletionListeners []NamespaceDeletionListener
}

// newNamespaceDispatcher creates and returns a new namespace handler.
func newNamespaceDispatcher(nsStore *namespaceStore, deletionListeners ...NamespaceDeletionListener) *namespaceDispatcher {
	return &namespaceDispatcher{
		nsStore:           nsStore,
		deletionListeners: deletionListeners,
	}
}

// ProcessEvent processes namespace resource events, and returns the sensor events to emit in response.
func (h *namespaceDispatcher) ProcessEvent(obj, _ interface{}, action central.ResourceAction) *component.ResourceEvent {
	ns := obj.(*v1.Namespace)

	roxNamespace := &storage.NamespaceMetadata{
		Id:           string(ns.GetUID()),
		Name:         ns.GetName(),
		Labels:       ns.GetLabels(),
		Annotations:  ns.GetAnnotations(),
		CreationTime: protoconv.ConvertTimeToTimestamp(ns.GetCreationTimestamp().Time),
	}

	if action == central.ResourceAction_REMOVE_RESOURCE {
		for _, listener := range h.deletionListeners {
			listener.OnNamespaceDeleted(ns.Name)
		}
		h.nsStore.removeNamespace(roxNamespace)
	} else {
		h.nsStore.addNamespace(roxNamespace)
	}

	return component.NewEvent(&central.SensorEvent{
		Id:     string(ns.GetUID()),
		Action: action,
		Resource: &central.SensorEvent_Namespace{
			Namespace: roxNamespace,
		},
	})
}
