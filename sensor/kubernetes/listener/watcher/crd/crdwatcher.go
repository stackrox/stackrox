package crd

import (
	"net/http"
	"sync/atomic"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/k8swatch"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/sensor/kubernetes/listener/watcher"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
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
	k8sClient *http.Client
	started   atomic.Bool
}

// NewCRDWatcher creates a new CRDWatcher
func NewCRDWatcher(stopSig *concurrency.Signal, k8sClient *http.Client) *crdWatcher {
	return &crdWatcher{
		stopSig:   stopSig,
		resources: set.NewStringSet(),
		k8sClient: k8sClient,
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

	// Use k8swatch adapter to watch CRDs instead of dynamic informer.
	// API path for CRDs is /apis/apiextensions.k8s.io/v1/customresourcedefinitions
	informer := k8swatch.NewInformerAdapter(
		"/apis/apiextensions.k8s.io/v1/customresourcedefinitions",
		w.k8sClient,
		func() runtime.Object { return &v1.CustomResourceDefinition{} },
	)

	_, err := informer.AddEventHandler(handler)
	if err != nil {
		return errors.Wrap(err, "adding CRD event handler")
	}

	// Start the adapter's watch goroutine
	go informer.Run(w.stopSig.Done())

	go watch(callback, w.resources.Freeze(), w.stopSig.Done(), w.resourceC)

	if !cache.WaitForCacheSync(w.stopSig.Done(), informer.HasSynced) {
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
