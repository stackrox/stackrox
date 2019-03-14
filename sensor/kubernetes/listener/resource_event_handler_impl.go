package listener

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/kubernetes/listener/resources"
)

// resourceEventHandlerImpl processes OnAdd, OnUpdate, and OnDelete events, and joins the results to an output
// channel
type resourceEventHandlerImpl struct {
	dispatcher            resources.Dispatcher
	output                chan<- *central.SensorEvent
	treatCreatesAsUpdates *concurrency.Flag
}

func (h *resourceEventHandlerImpl) OnAdd(obj interface{}) {
	// If we are listing the initial objects, then we treat them as updates so enforcement isn't done
	if h.treatCreatesAsUpdates != nil && h.treatCreatesAsUpdates.Get() {
		h.sendResourceEvent(obj, central.ResourceAction_UPDATE_RESOURCE)
	} else {
		h.sendResourceEvent(obj, central.ResourceAction_CREATE_RESOURCE)
	}
}

func (h *resourceEventHandlerImpl) OnUpdate(oldObj, newObj interface{}) {
	h.sendResourceEvent(newObj, central.ResourceAction_UPDATE_RESOURCE)
}

func (h *resourceEventHandlerImpl) OnDelete(obj interface{}) {
	h.sendResourceEvent(obj, central.ResourceAction_REMOVE_RESOURCE)
}

func (h *resourceEventHandlerImpl) sendResourceEvent(obj interface{}, action central.ResourceAction) {
	evWraps := h.dispatcher.ProcessEvent(obj, action)
	h.sendEvents(evWraps...)
}

func (h *resourceEventHandlerImpl) sendEvents(evWraps ...*central.SensorEvent) {
	for _, evWrap := range evWraps {
		h.output <- evWrap
	}
}
