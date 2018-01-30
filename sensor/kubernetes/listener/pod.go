package listener

import (
	"time"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

type podLister interface {
	list(map[string]string) []v1.Pod
}

// podWatchLister watches and lists all pods.
type podWatchLister struct {
	client     rest.Interface
	store      cache.Store
	controller cache.Controller
	stopC      chan struct{}
}

func newPodWatchLister(client rest.Interface) *podWatchLister {
	return &podWatchLister{
		client: client,
		stopC:  make(chan struct{}),
	}
}

func (wl *podWatchLister) watch() {
	watchlist := cache.NewListWatchFromClient(wl.client, "pods", v1.NamespaceAll, fields.Everything())

	wl.store, wl.controller = cache.NewInformer(
		watchlist,
		&v1.Pod{},
		resyncPeriod,
		cache.ResourceEventHandlerFuncs{},
	)

	wl.controller.Run(wl.stopC)
}

func (wl *podWatchLister) list(labelSelector map[string]string) (pods []v1.Pod) {
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

func (wl *podWatchLister) blockUntilSynced() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		if wl.controller.HasSynced() {
			return
		}
	}
}

func (wl *podWatchLister) stop() {
	wl.stopC <- struct{}{}
}
