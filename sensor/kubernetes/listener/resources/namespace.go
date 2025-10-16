package resources

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	"google.golang.org/protobuf/proto"
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

	roxNamespace := &storage.NamespaceMetadata{}
	roxNamespace.SetId(string(ns.GetUID()))
	roxNamespace.SetName(ns.GetName())
	roxNamespace.SetLabels(ns.GetLabels())
	roxNamespace.SetAnnotations(ns.GetAnnotations())
	roxNamespace.SetCreationTime(protoconv.ConvertTimeToTimestamp(ns.GetCreationTimestamp().Time))

	if action == central.ResourceAction_REMOVE_RESOURCE {
		for _, listener := range h.deletionListeners {
			listener.OnNamespaceDeleted(ns.Name)
		}
		h.nsStore.removeNamespace(roxNamespace)
	} else {
		h.nsStore.addNamespace(roxNamespace)
	}

	se := &central.SensorEvent{}
	se.SetId(string(ns.GetUID()))
	se.SetAction(action)
	se.SetNamespace(proto.ValueOrDefault(roxNamespace))
	return component.NewEvent(se)
}
