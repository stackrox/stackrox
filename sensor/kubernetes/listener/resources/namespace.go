package resources

import (
	pkgV1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"k8s.io/api/core/v1"
)

// NamespaceDeletionListener allows components to react to the deletion of namespaces.
type NamespaceDeletionListener interface {
	OnNamespaceDeleted(string)
}

// namespaceHandler handles namespace resource events.
type namespaceHandler struct {
	deletionListeners []NamespaceDeletionListener
}

// newNamespaceHandler creates and returns a new namespace handler.
func newNamespaceHandler(deletionListeners ...NamespaceDeletionListener) *namespaceHandler {
	return &namespaceHandler{
		deletionListeners: deletionListeners,
	}
}

// Process processes a namespace resource events, and returns the sensor events to emit in response.
func (h *namespaceHandler) Process(ns *v1.Namespace, action pkgV1.ResourceAction) []*pkgV1.SensorEvent {
	if action == pkgV1.ResourceAction_REMOVE_RESOURCE {
		for _, listener := range h.deletionListeners {
			listener.OnNamespaceDeleted(ns.Name)
		}
	}

	return []*pkgV1.SensorEvent{{
		Id:     string(ns.GetUID()),
		Action: action,
		Resource: &pkgV1.SensorEvent_Namespace{
			Namespace: &storage.Namespace{
				Id:     string(ns.GetUID()),
				Name:   ns.GetName(),
				Labels: ns.GetLabels(),
			},
		},
	},
	}
}
