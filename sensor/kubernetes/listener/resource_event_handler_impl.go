package listener

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/kubernetes/listener/resources"
	"k8s.io/apimachinery/pkg/types"
)

// resourceEventHandlerImpl processes OnAdd, OnUpdate, and OnDelete events, and joins the results to an output
// channel
type resourceEventHandlerImpl struct {
	dispatcher            resources.Dispatcher
	output                chan<- *central.SensorEvent
	treatCreatesAsUpdates *concurrency.Flag

	syncLock                   sync.Mutex
	seenIDs                    map[types.UID]struct{}
	missingInitialIDs          map[types.UID]struct{}
	hasSeenAllInitialIDsSignal concurrency.Signal
}

func (h *resourceEventHandlerImpl) OnAdd(obj interface{}) {
	// If we are listing the initial objects, then we treat them as updates so enforcement isn't done
	if h.treatCreatesAsUpdates != nil && h.treatCreatesAsUpdates.Get() {
		h.sendResourceEvent(obj, central.ResourceAction_UPDATE_RESOURCE)
	} else {
		h.sendResourceEvent(obj, central.ResourceAction_CREATE_RESOURCE)
	}
	h.registerObject(obj)
}

func (h *resourceEventHandlerImpl) OnUpdate(oldObj, newObj interface{}) {
	h.sendResourceEvent(newObj, central.ResourceAction_UPDATE_RESOURCE)
	h.registerObject(newObj)
}

func (h *resourceEventHandlerImpl) OnDelete(obj interface{}) {
	h.sendResourceEvent(obj, central.ResourceAction_REMOVE_RESOURCE)
}

func (h *resourceEventHandlerImpl) PopulateInitialObjects(initialObjs []interface{}) <-chan struct{} {
	h.populateInitialObjects(initialObjs)
	return h.hasSeenAllInitialIDsSignal.Done()
}

func (h *resourceEventHandlerImpl) populateInitialObjects(initialObjs []interface{}) {
	if h.hasSeenAllInitialIDsSignal.IsDone() {
		return
	}

	h.syncLock.Lock()
	defer h.syncLock.Unlock()
	if h.seenIDs == nil {
		return
	}
	h.missingInitialIDs = make(map[types.UID]struct{})
	for _, obj := range initialObjs {
		newUID := getObjUID(obj)
		if _, ok := h.seenIDs[newUID]; !ok {
			h.missingInitialIDs[newUID] = struct{}{}
		}
	}
	h.seenIDs = nil
	h.checkHasSeenAllInitialIDsNoLock()
}

func (h *resourceEventHandlerImpl) registerObject(newObj interface{}) {
	if h.hasSeenAllInitialIDsSignal.IsDone() {
		return
	}

	newUID := getObjUID(newObj)
	h.syncLock.Lock()
	defer h.syncLock.Unlock()
	if h.seenIDs != nil {
		h.seenIDs[newUID] = struct{}{}
	} else if h.missingInitialIDs != nil {
		delete(h.missingInitialIDs, newUID)
		h.checkHasSeenAllInitialIDsNoLock()
	}
}

func (h *resourceEventHandlerImpl) checkHasSeenAllInitialIDsNoLock() {
	if h.missingInitialIDs != nil && len(h.missingInitialIDs) == 0 {
		h.missingInitialIDs = nil
		h.hasSeenAllInitialIDsSignal.Signal()
	}
}

func (h *resourceEventHandlerImpl) sendResourceEvent(obj interface{}, action central.ResourceAction) {
	evWraps := h.dispatcher.ProcessEvent(obj, action)
	h.sendEvents(evWraps...)
}

func (h *resourceEventHandlerImpl) sendEvents(evWraps ...*central.SensorEvent) {
	for _, evWrap := range evWraps {
		h.output <- protoutils.CloneCentralSensorEvent(evWrap)
	}
}

func getObjUID(newObj interface{}) types.UID {
	if objWithID, ok := newObj.(interface{ GetUID() types.UID }); ok {
		return objWithID.GetUID()
	}

	errorhelpers.PanicOnDevelopmentf("this object didn't have an ID %T, %+v", newObj, newObj)
	return ""
}
