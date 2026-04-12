package k8swatch

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
)

// InformerAdapter wraps a minimal watcher to satisfy enough of the
// cache.SharedIndexInformer interface to work with sensor's handle() function.
// It does NOT maintain a cache — events are forwarded immediately.
type InformerAdapter struct {
	apiPath    string
	client     *http.Client
	newObject  func() runtime.Object
	handlers   []cache.ResourceEventHandler
	handlerMu  sync.RWMutex
	hasSynced  concurrency.Signal
	cancelFunc context.CancelFunc
}

// NewInformerAdapter creates an adapter that watches the given API path and
// deserializes events into typed objects using newObject as a factory.
func NewInformerAdapter(apiPath string, client *http.Client, newObject func() runtime.Object) *InformerAdapter {
	return &InformerAdapter{
		apiPath:   apiPath,
		client:    client,
		newObject: newObject,
		hasSynced: concurrency.NewSignal(),
	}
}

// AddEventHandler implements cache.SharedInformer.
func (a *InformerAdapter) AddEventHandler(handler cache.ResourceEventHandler) (cache.ResourceEventHandlerRegistration, error) {
	a.handlerMu.Lock()
	defer a.handlerMu.Unlock()
	a.handlers = append(a.handlers, handler)
	return nil, nil
}

// HasSynced implements cache.SharedInformer.
func (a *InformerAdapter) HasSynced() bool {
	return a.hasSynced.IsDone()
}

// Run starts the watch. Called by the informer factory's Start().
func (a *InformerAdapter) Run(stopCh <-chan struct{}) {
	ctx, cancel := context.WithCancel(context.Background())
	a.cancelFunc = cancel

	go func() {
		<-stopCh
		cancel()
	}()

	// Mark as synced after first successful event or connection.
	// NOTE: Unlike informers, we don't do LIST+WATCH — we start with WATCH only.
	// This means we won't see objects that existed before sensor started.
	// The initial LIST is handled by Central's reconciliation on connect.
	firstEvent := sync.Once{}
	log.Infof("k8swatch adapter: starting for %s (handlers=%d)", a.apiPath, len(a.handlers))

	watcher := New(a.apiPath, a.client, func(eventType string, raw json.RawMessage) {
		firstEvent.Do(func() {
			log.Infof("k8swatch adapter: %s first event received, marking synced", a.apiPath)
			a.hasSynced.Signal()
		})

		obj := a.newObject()
		if err := json.Unmarshal(raw, obj); err != nil {
			log.Warnf("k8swatch adapter %s: failed to unmarshal %s event (%d bytes): %v",
				a.apiPath, eventType, len(raw), err)
			// Log a snippet of the raw JSON for debugging
			snippet := string(raw)
			if len(snippet) > 200 {
				snippet = snippet[:200] + "..."
			}
			log.Debugf("k8swatch adapter %s: raw JSON: %s", a.apiPath, snippet)
			return
		}

		a.handlerMu.RLock()
		defer a.handlerMu.RUnlock()

		for _, handler := range a.handlers {
			switch eventType {
			case "ADDED":
				handler.OnAdd(obj, false)
			case "MODIFIED":
				handler.OnUpdate(nil, obj) // oldObj is nil — dispatchers must handle this
			case "DELETED":
				handler.OnDelete(obj)
			case "ERROR":
				log.Warnf("k8swatch adapter %s: received ERROR event: %s", a.apiPath, string(raw))
			default:
				log.Warnf("k8swatch adapter %s: unknown event type %q", a.apiPath, eventType)
			}
		}
	})

	watcher.Run(ctx)
}

// Stubs for cache.SharedIndexInformer interface methods we don't use.
// These exist so InformerAdapter satisfies the interface and can be
// plugged into existing code that expects a SharedIndexInformer.
func (a *InformerAdapter) AddEventHandlerWithResyncPeriod(handler cache.ResourceEventHandler, _ time.Duration) (cache.ResourceEventHandlerRegistration, error) {
	return a.AddEventHandler(handler)
}
func (a *InformerAdapter) AddEventHandlerWithOptions(handler cache.ResourceEventHandler, _ cache.HandlerOptions) (cache.ResourceEventHandlerRegistration, error) {
	return a.AddEventHandler(handler)
}
func (a *InformerAdapter) RemoveEventHandler(_ cache.ResourceEventHandlerRegistration) error {
	return nil
}
func (a *InformerAdapter) GetStore() cache.Store                                { return nil }
func (a *InformerAdapter) GetController() cache.Controller                      { return nil }
func (a *InformerAdapter) LastSyncResourceVersion() string                      { return "" }
func (a *InformerAdapter) SetWatchErrorHandler(_ cache.WatchErrorHandler) error { return nil }
func (a *InformerAdapter) SetWatchErrorHandlerWithContext(_ cache.WatchErrorHandlerWithContext) error {
	return nil
}
func (a *InformerAdapter) SetTransform(_ cache.TransformFunc) error { return nil }
func (a *InformerAdapter) IsStopped() bool                          { return false }
func (a *InformerAdapter) AddIndexers(_ cache.Indexers) error       { return nil }
func (a *InformerAdapter) GetIndexer() cache.Indexer                { return nil }
func (a *InformerAdapter) RunWithContext(_ context.Context)         { /* Run() handles lifecycle */ }
func (a *InformerAdapter) String() string {
	return fmt.Sprintf("k8swatch.InformerAdapter{%s}", a.apiPath)
}
