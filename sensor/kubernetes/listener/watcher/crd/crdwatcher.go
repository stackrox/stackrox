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

type crdWatcher struct {
	stopSig            *concurrency.Signal
	resources          set.StringSet
	availableResources set.StringSet
	resourceC          <-chan *resourceEvent
	sif                dynamicinformer.DynamicSharedInformerFactory
	status             bool
	started            *atomic.Bool
}

// NewCRDWatcher creates a new CRDWatcher
func NewCRDWatcher(stopSig *concurrency.Signal, sif dynamicinformer.DynamicSharedInformerFactory) *crdWatcher {
	return &crdWatcher{
		stopSig:            stopSig,
		resources:          set.NewStringSet(),
		availableResources: set.NewStringSet(),
		sif:                sif,
		status:             false,
		started:            &atomic.Bool{},
	}
}

type resourceEvent struct {
	resourceName string
	action       central.ResourceAction
}

func (w *crdWatcher) startHandler() error {
	eventC, handler := newCRDHandler(w.stopSig)
	w.resourceC = eventC
	informer := w.sif.ForResource(v1.SchemeGroupVersion.WithResource(customResourceDefinitionsName)).Informer()
	if _, err := informer.AddEventHandler(handler); err != nil {
		return errors.Wrap(err, "adding CRD event handler")
	}
	w.sif.Start(w.stopSig.Done())
	if !cache.WaitForCacheSync(w.stopSig.Done(), informer.HasSynced) {
		log.Warn("Failed to wait for informer cache sync")
	}
	return nil
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
func (w *crdWatcher) Watch() (<-chan *watcher.Status, error) {
	if w.started.Swap(true) {
		return nil, errors.New("Watch was already called")
	}
	err := w.startHandler()
	statusC := make(chan *watcher.Status)
	go func() {
		defer close(statusC)
		for {
			select {
			case <-w.stopSig.Done():
				return
			case event, ok := <-w.resourceC:
				if !ok {
					return
				}
				if !w.resources.Contains(event.resourceName) {
					continue
				}
				switch event.action {
				case central.ResourceAction_CREATE_RESOURCE:
					w.availableResources.Add(event.resourceName)
				case central.ResourceAction_REMOVE_RESOURCE:
					w.availableResources.Remove(event.resourceName)
				}
			}
			if (len(w.resources) == len(w.availableResources) && !w.status) ||
				(len(w.resources) > len(w.availableResources) && w.status) {
				w.status = !w.status
				select {
				case <-w.stopSig.Done():
					return
				case statusC <- &watcher.Status{
					Available: w.status,
					Resources: w.resources,
				}:
				}
			}
		}
	}()
	return statusC, err
}
