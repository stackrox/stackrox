package listener

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/kubernetes"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	"github.com/stackrox/rox/sensor/kubernetes/listener/resources"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
)

// resourceEventHandlerImpl processes OnAdd, OnUpdate, and OnDelete events, and joins the results to an output
// channel
type resourceEventHandlerImpl struct {
	name       string
	context    context.Context
	eventLock  *sync.Mutex
	dispatcher resources.Dispatcher

	resolver         component.Resolver
	syncingResources *concurrency.Flag

	syncLock                   sync.Mutex
	seenIDs                    map[types.UID]struct{}
	missingInitialIDs          map[types.UID]struct{}
	hasSeenAllInitialIDsSignal concurrency.Signal
}

func (h *resourceEventHandlerImpl) OnAdd(obj interface{}, _ bool) {
	// If we are listing the initial objects, then we treat them as updates so enforcement isn't done
	if h.syncingResources != nil && h.syncingResources.Get() {
		h.sendResourceEvent(obj, nil, central.ResourceAction_SYNC_RESOURCE)
	} else {
		h.sendResourceEvent(obj, nil, central.ResourceAction_CREATE_RESOURCE)
	}
	h.registerObject(obj)
}

func (h *resourceEventHandlerImpl) OnUpdate(oldObj, newObj interface{}) {
	h.sendResourceEvent(newObj, oldObj, central.ResourceAction_UPDATE_RESOURCE)
	h.registerObject(newObj)
}

func (h *resourceEventHandlerImpl) OnDelete(obj interface{}) {
	if tombstone, ok := obj.(cache.DeletedFinalStateUnknown); ok {
		obj = tombstone.Obj // we don't care about the final state, so just using the last known state is fine.
	}
	h.sendResourceEvent(obj, nil, central.ResourceAction_REMOVE_RESOURCE)
}

func (h *resourceEventHandlerImpl) PopulateInitialObjects(initialObjs []interface{}) <-chan struct{} {
	h.populateInitialObjects(initialObjs)
	return h.hasSeenAllInitialIDsSignal.Done()
}

func (h *resourceEventHandlerImpl) populateInitialObjects(initialObjs []interface{}) {
	var isSecret bool
	if len(initialObjs) > 0 {
		for i, obj := range initialObjs {
			if secret, ok := obj.(*corev1.Secret); ok {
				isSecret = true
				log.Debugf("ROX-24163 populateInitialObjects processes secret no %d ID=%q type=%q",
					i, secret.GetUID(), secret.Type)
			}
		}
	}

	if h.hasSeenAllInitialIDsSignal.IsDone() {
		if isSecret {
			log.Debugf("ROX-24163 populateInitialObjects hasSeenAllInitialIDsSignal is done")
		}
		return
	}

	h.syncLock.Lock()
	defer h.syncLock.Unlock()
	if h.seenIDs == nil {
		if isSecret {
			log.Debugf("ROX-24163 populateInitialObjects seenIDs is nil")
		}
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
	if isSecret {
		log.Debug("ROX-24163 populateInitialObjects calling checkHasSeenAllInitialIDsNoLock")
		defer log.Debug("ROX-24163 populateInitialObjects finished processesing secret")
	}
	h.checkHasSeenAllInitialIDsNoLock()
}

func (h *resourceEventHandlerImpl) registerObject(newObj interface{}) {
	if h.hasSeenAllInitialIDsSignal.IsDone() {
		return
	}

	secret, isSecret := newObj.(*corev1.Secret)
	if isSecret {
		log.Debugf("ROX-24163 registerObject processes secret (ns/name=%s/%s ID=%q type=%q)",
			secret.GetNamespace(), secret.GetName(), secret.GetUID(), secret.Type)
	}

	newUID := getObjUID(newObj)
	h.syncLock.Lock()
	defer h.syncLock.Unlock()
	if h.seenIDs != nil {
		h.seenIDs[newUID] = struct{}{}
	} else if h.missingInitialIDs != nil {
		if isSecret {
			log.Debugf("ROX-24163 registerObject deleting secret (ns/name=%s/%s ID=%q type=%q) form missingInitialIDs",
				secret.GetNamespace(), secret.GetName(), secret.GetUID(), secret.Type)
		}
		delete(h.missingInitialIDs, newUID)
		h.checkHasSeenAllInitialIDsNoLock()
	} else {
		if isSecret {
			log.Debugf("ROX-24163 registerObject else-case secret (ns/name=%s/%s ID=%q type=%q)",
				secret.GetNamespace(), secret.GetName(), secret.GetUID(), secret.Type)
		}
	}
	if isSecret {
		log.Debugf("ROX-24163 registerObject done processing secret (ns/name=%s/%s ID=%q type=%q). len(h.missingInitialIDs)=%d",
			secret.GetNamespace(), secret.GetName(), secret.GetUID(), secret.Type, len(h.missingInitialIDs))
	}
}

func (h *resourceEventHandlerImpl) checkHasSeenAllInitialIDsNoLock() {
	if h.missingInitialIDs != nil && len(h.missingInitialIDs) == 0 {
		h.missingInitialIDs = nil
		h.hasSeenAllInitialIDsSignal.Signal()
	}
}

func (h *resourceEventHandlerImpl) sendResourceEvent(obj, oldObj interface{}, action central.ResourceAction) {
	if metaObj, ok := obj.(v1.Object); ok {
		kubernetes.TrimAnnotations(metaObj)
	}

	message := h.dispatcher.ProcessEvent(obj, oldObj, action)
	message.Context = h.context
	h.resolver.Send(message)
}

func getObjUID(newObj interface{}) types.UID {
	if objWithID, ok := newObj.(interface{ GetUID() types.UID }); ok {
		return objWithID.GetUID()
	}

	utils.Should(errors.Errorf("this object didn't have an ID %T", newObj))
	return ""
}
