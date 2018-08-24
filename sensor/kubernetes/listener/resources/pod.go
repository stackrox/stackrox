package resources

import (
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

type podLister interface {
	List(map[string]string) []v1.Pod
}

// PodWatchLister watches and lists all pods.
type PodWatchLister struct {
	client       rest.Interface
	store        cache.Store
	controller   cache.Controller
	stopSig      concurrency.Signal
	resyncPeriod time.Duration
}

// NewPodWatchLister implements the WatchLister for pods
func NewPodWatchLister(client rest.Interface, resyncPeriod time.Duration) *PodWatchLister {
	return &PodWatchLister{
		client:       client,
		stopSig:      concurrency.NewSignal(),
		resyncPeriod: resyncPeriod,
	}
}

// Watch starts the watch
func (wl *PodWatchLister) Watch() {
	watchlist := cache.NewListWatchFromClient(wl.client, "pods", v1.NamespaceAll, fields.Everything())

	wl.store, wl.controller = cache.NewInformer(
		watchlist,
		&v1.Pod{},
		wl.resyncPeriod,
		cache.ResourceEventHandlerFuncs{},
	)

	wl.controller.Run(wl.stopSig.Done())
}

// List lists all of the pods
func (wl *PodWatchLister) List(labelSelector map[string]string) (pods []v1.Pod) {
	err := cache.ListAll(wl.store, labels.Set(labelSelector).AsSelector(), func(obj interface{}) {
		if p, ok := obj.(*v1.Pod); ok {
			pods = append(pods, *p)
		} else {
			logger.Errorf("obj %+v is not of type pods", obj)
		}
	})

	if err != nil {
		logger.Errorf("unable to list pods: %s", err)
		return []v1.Pod{}
	}

	return
}

// BlockUntilSynced waits for the watch controller to be synced
func (wl *PodWatchLister) BlockUntilSynced() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		if wl.controller.HasSynced() {
			return
		}
	}
}

// Stop stops the watch
func (wl *PodWatchLister) Stop() {
	wl.stopSig.Signal()
}
