package crd

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type crdHandler struct {
	stopSig *concurrency.Signal
	eventC  chan *resourceEvent
}

func newCRDHandler(stopSig *concurrency.Signal, eventC chan *resourceEvent) *crdHandler {
	return &crdHandler{
		stopSig: stopSig,
		eventC:  eventC,
	}
}

// OnAdd this function is called by the informer whenever a resource is created in the cluster
func (h *crdHandler) OnAdd(obj interface{}, _ bool) {
	h.processEvent(nil, obj, central.ResourceAction_CREATE_RESOURCE)
}

// OnUpdate this function is called by the informer whenever a resource is updated in the cluster
func (h *crdHandler) OnUpdate(_, _ interface{}) {}

// OnDelete this function is called by the informer whenever a resource is deleted the cluster
func (h *crdHandler) OnDelete(obj interface{}) {
	h.processEvent(nil, obj, central.ResourceAction_REMOVE_RESOURCE)
}

func (h *crdHandler) processEvent(_, new interface{}, action central.ResourceAction) {
	unstructuredObj, ok := new.(*unstructured.Unstructured)
	if !ok {
		log.Errorf("Object %v is not an unstructured object", new)
		return
	}
	log.Debugf("Process CRD %s with action %s", unstructuredObj.GetName(), action)
	select {
	case <-h.stopSig.Done():
		return
	case h.eventC <- &resourceEvent{
		resourceName: unstructuredObj.GetName(),
		action:       action,
	}:
	}
}
