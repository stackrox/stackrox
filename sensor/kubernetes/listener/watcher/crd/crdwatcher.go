package crd

import (
	"sync/atomic"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/sensor/kubernetes/listener/watcher"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
)

const (
	customResourceDefinitionsName = "customresourcedefinitions"
)

var (
	log = logging.LoggerForModule()
)

type WatcherCallback func(*watcher.Status)

type crdWatcher struct {
	stopSig   *concurrency.Signal
	resources set.StringSet
	resourceC <-chan *resourceEvent
	sif       dynamicinformer.DynamicSharedInformerFactory
	started   atomic.Bool
}

// NewCRDWatcher creates a new CRDWatcher
func NewCRDWatcher(stopSig *concurrency.Signal, sif dynamicinformer.DynamicSharedInformerFactory) *crdWatcher {
	return &crdWatcher{
		stopSig:   stopSig,
		resources: set.NewStringSet(),
		sif:       sif,
		started:   atomic.Bool{},
	}
}

type resourceEvent struct {
	resourceName string
	action       central.ResourceAction
}

// AddResourceToWatch adds a resource to be watched
func (w *crdWatcher) AddResourceToWatch(name string) error {
	if w.started.Load() {
		return errors.New("Adding resources to watch after calling 'Watch' is not supported")
	}
	w.resources.Add(name)
	return nil
}

// Watch starts the CRD handler that will dispatch any events coming from k8s related to CRDs to be manage by the CRDWatcher
func (w *crdWatcher) Watch(callback WatcherCallback) error {
	if w.started.Swap(true) {
		return errors.New("Watch was already called")
	}

	eventC := make(chan *resourceEvent)
	handler := &crdHandler{
		stopSig: w.stopSig,
		eventC:  eventC,
	}
	w.resourceC = eventC
	informer := w.sif.ForResource(v1.SchemeGroupVersion.WithResource(customResourceDefinitionsName)).Informer()
	h, err := informer.AddEventHandler(handler)
	if err != nil {
		return errors.Wrap(err, "adding CRD event handler")
	}
	w.sif.Start(w.stopSig.Done())

	go watch(callback, w.resources.Freeze(), w.stopSig.Done(), w.resourceC)

	if !cache.WaitForCacheSync(w.stopSig.Done(), h.HasSynced) {
		log.Warn("Failed to wait for handler cache sync")
	}

	return nil
}

func watch(callback WatcherCallback, resources set.FrozenSet[string], done <-chan struct{}, resourceC <-chan *resourceEvent) {
	previousStatus := false
	resourcesCount := resources.Cardinality()
	availableResources := make(set.StringSet, resourcesCount)
	for {
		select {
		case <-done:
			return
		case event, ok := <-resourceC:
			if !ok {
				return
			}
			if !resources.Contains(event.resourceName) {
				continue
			}
			switch event.action {
			case central.ResourceAction_CREATE_RESOURCE:
				availableResources.Add(event.resourceName)
			case central.ResourceAction_REMOVE_RESOURCE:
				availableResources.Remove(event.resourceName)
			}
		}
		// Send status only when availability changes.
		// If we reach resourcesCount and previousStatus state was not available,
		// or it was available but element was removed.
		currentStatus := resourcesCount == len(availableResources)
		if currentStatus == previousStatus {
			continue
		}
		callback(&watcher.Status{
			Available: currentStatus,
			Resources: resources,
		})
		previousStatus = currentStatus
	}
}
