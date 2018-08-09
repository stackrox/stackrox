package watchlister

import (
	"time"

	pkgV1 "github.com/stackrox/rox/generated/api/v1"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

// WatchLister is the generic watcher for k8s
type WatchLister struct {
	client     rest.Interface
	Store      cache.Store
	Controller cache.Controller
	stopC      chan struct{}

	resyncPeriod time.Duration
}

// NewWatchLister instantiates a new generic lister
func NewWatchLister(client rest.Interface, resyncPeriod time.Duration) WatchLister {
	return WatchLister{
		client:       client,
		stopC:        make(chan struct{}),
		resyncPeriod: resyncPeriod,
	}
}

// SetupWatch initializes the client
func (wl *WatchLister) SetupWatch(object string, objectType runtime.Object, changedFunc func(interface{}, pkgV1.ResourceAction)) {
	watchlist := cache.NewListWatchFromClient(wl.client, object, v1.NamespaceAll, fields.Everything())

	wl.Store, wl.Controller = cache.NewInformer(
		watchlist,
		objectType,
		wl.resyncPeriod,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				// Once the initial objects are listed, the resource action changes to CREATE.
				changedFunc(obj, pkgV1.ResourceAction_PREEXISTING_RESOURCE)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				changedFunc(newObj, pkgV1.ResourceAction_UPDATE_RESOURCE)
			},
			DeleteFunc: func(obj interface{}) {
				changedFunc(obj, pkgV1.ResourceAction_REMOVE_RESOURCE)
			},
		},
	)
}

// StartWatch starts watching
func (wl *WatchLister) StartWatch() {
	wl.Controller.Run(wl.stopC)
}

// Stop stops the watch
func (wl *WatchLister) Stop() {
	wl.stopC <- struct{}{}
}
