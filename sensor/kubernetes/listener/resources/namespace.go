package resources

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

// NamespaceDeletionListener allows components to react to the deletion of namespaces.
type NamespaceDeletionListener interface {
	OnNamespaceDeleted(string)
}

// namespaceDispatcher handles namespace resource events.
type namespaceDispatcher struct {
	deletionListeners []NamespaceDeletionListener

	k8sClient kubernetes.Clientset
}

// newNamespaceDispatcher creates and returns a new namespace handler.
func newNamespaceDispatcher(deletionListeners ...NamespaceDeletionListener) *namespaceDispatcher {
	return &namespaceDispatcher{
		deletionListeners: deletionListeners,
	}
}

// Process processes a namespace resource events, and returns the sensor events to emit in response.
func (h *namespaceDispatcher) ProcessEvent(obj interface{}, action central.ResourceAction) []*central.SensorEvent {
	ns := obj.(*v1.Namespace)

	if action == central.ResourceAction_REMOVE_RESOURCE {
		for _, listener := range h.deletionListeners {
			listener.OnNamespaceDeleted(ns.Name)
		}
	}

	return []*central.SensorEvent{{
		Id:     string(ns.GetUID()),
		Action: action,
		Resource: &central.SensorEvent_Namespace{
			Namespace: &storage.Namespace{
				Id:     string(ns.GetUID()),
				Name:   ns.GetName(),
				Labels: ns.GetLabels(),
			},
		},
	},
	}
}
