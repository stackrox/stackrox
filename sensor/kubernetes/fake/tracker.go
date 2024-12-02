package fake

import (
	"sync"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/testing"
)

type tracker struct {
	scheme  testing.ObjectScheme
	decoder runtime.Decoder
	lock    sync.RWMutex
	objects map[schema.GroupVersionResource]map[types.NamespacedName]runtime.Object
	// The value type of watchers is a map of which the key is either a namespace or
	// all/non namespace aka "" and its value is list of fake watchers.
	// Manipulations on resources will broadcast the notification events into the
	// watchers' channel. Note that too many unhandled events (currently 100,
	// see apimachinery/pkg/watch.DefaultChanSize) will cause a panic.
	watchers      map[schema.GroupVersionResource]map[string][]*watch.RaceFreeFakeWatcher
	watchCounter  int
	addCounter    int
	deleteCounter int
}

var _ testing.ObjectTracker = &tracker{}

// NewObjectTracker returns an ObjectTracker that can be used to keep track
// of objects for the fake clientset. Mostly useful for unit tests.
func NewObjectTracker(scheme testing.ObjectScheme, decoder runtime.Decoder) tracker {
	return tracker{
		scheme:        scheme,
		decoder:       decoder,
		objects:       make(map[schema.GroupVersionResource]map[types.NamespacedName]runtime.Object),
		watchers:      make(map[schema.GroupVersionResource]map[string][]*watch.RaceFreeFakeWatcher),
		watchCounter:  0,
		addCounter:    0,
		deleteCounter: 0,
	}
}

func (t *tracker) List(gvr schema.GroupVersionResource, gvk schema.GroupVersionKind, ns string) (runtime.Object, error) {
	return nil, nil
}

func (t *tracker) Watch(gvr schema.GroupVersionResource, ns string) (watch.Interface, error) {
	t.lock.Lock()
	defer t.lock.Unlock()
	t.watchCounter++
	return nil, nil
}

func (t *tracker) Get(gvr schema.GroupVersionResource, ns, name string) (runtime.Object, error) {
	return nil, nil
}

func (t *tracker) Add(obj runtime.Object) error {
	t.lock.Lock()
	defer t.lock.Unlock()
	t.addCounter++
	return nil
}

func (t *tracker) Create(gvr schema.GroupVersionResource, obj runtime.Object, ns string) error {
	t.lock.Lock()
	defer t.lock.Unlock()
	t.addCounter++
	return nil
}

func (t *tracker) Update(gvr schema.GroupVersionResource, obj runtime.Object, ns string) error {
	return nil
}

func (t *tracker) getWatches(gvr schema.GroupVersionResource, ns string) []*watch.RaceFreeFakeWatcher {
	return nil
}

func (t *tracker) add(gvr schema.GroupVersionResource, obj runtime.Object, ns string, replaceExisting bool) error {
	return nil
}

func (t *tracker) addList(obj runtime.Object, replaceExisting bool) error {
	return nil
}

func (t *tracker) Delete(gvr schema.GroupVersionResource, ns, name string) error {
	t.lock.Lock()
	defer t.lock.Unlock()
	t.deleteCounter++
	return nil
}

// GetCounts returns watchCount, addCount, deleteCount
func (t *tracker) GetCounts() (int, int, int) {
	return t.watchCounter, t.addCounter, t.deleteCounter
}
