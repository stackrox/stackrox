package k8swatch

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
)

// testHandler records events for verification.
type testHandler struct {
	mu      sync.Mutex
	added   []string
	updated []string
	deleted []string
}

func (h *testHandler) OnAdd(obj interface{}, _ bool) {
	if node, ok := obj.(*corev1.Node); ok {
		h.mu.Lock()
		defer h.mu.Unlock()
		h.added = append(h.added, node.Name)
	}
}

func (h *testHandler) OnUpdate(_, newObj interface{}) {
	if node, ok := newObj.(*corev1.Node); ok {
		h.mu.Lock()
		defer h.mu.Unlock()
		h.updated = append(h.updated, node.Name)
	}
}

func (h *testHandler) OnDelete(obj interface{}) {
	if node, ok := obj.(*corev1.Node); ok {
		h.mu.Lock()
		defer h.mu.Unlock()
		h.deleted = append(h.deleted, node.Name)
	}
}

func (h *testHandler) getAdded() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	result := make([]string, len(h.added))
	copy(result, h.added)
	return result
}

func (h *testHandler) getUpdated() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	result := make([]string, len(h.updated))
	copy(result, h.updated)
	return result
}

func (h *testHandler) getDeleted() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	result := make([]string, len(h.deleted))
	copy(result, h.deleted)
	return result
}

func newNodeJSON(name string) string {
	return fmt.Sprintf(`{"apiVersion":"v1","kind":"Node","metadata":{"name":%q,"resourceVersion":"1"}}`, name)
}

func TestAdapter_InitialList(t *testing.T) {
	// Server that handles LIST (no watch param) and WATCH
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("watch") != "true" {
			// LIST response
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"items":[%s,%s]}`, newNodeJSON("node-1"), newNodeJSON("node-2"))
			return
		}
		// WATCH — just stay open until client disconnects
		<-r.Context().Done()
	}))
	defer server.Close()

	adapter := NewInformerAdapterForTest(server.URL, "/api/v1/nodes", server.Client(), func() runtime.Object {
		return &corev1.Node{}
	})

	handler := &testHandler{}
	_, err := adapter.AddEventHandler(handler)
	require.NoError(t, err)

	stopCh := make(chan struct{})

	// Run in goroutine since it blocks
	go adapter.Run(stopCh)

	// Wait for sync
	require.Eventually(t, func() bool {
		return adapter.HasSynced()
	}, 5*time.Second, 100*time.Millisecond, "adapter should sync after initial list")

	close(stopCh)

	added := handler.getAdded()
	assert.Len(t, added, 2)
	assert.Contains(t, added, "node-1")
	assert.Contains(t, added, "node-2")
}

func TestAdapter_WatchEventsAfterList(t *testing.T) {
	eventSent := make(chan struct{})

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("watch") != "true" {
			// Empty LIST
			fmt.Fprint(w, `{"items":[]}`)
			return
		}
		// WATCH — send one event
		flusher, _ := w.(http.Flusher)
		event := fmt.Sprintf(`{"type":"ADDED","object":%s}`, newNodeJSON("new-node"))
		fmt.Fprintln(w, event)
		flusher.Flush()
		close(eventSent)
		// Keep connection open
		<-r.Context().Done()
	}))
	defer server.Close()

	adapter := NewInformerAdapterForTest(server.URL, "/api/v1/nodes", server.Client(), func() runtime.Object {
		return &corev1.Node{}
	})

	handler := &testHandler{}
	adapter.AddEventHandler(handler)

	stopCh := make(chan struct{})
	go adapter.Run(stopCh)

	// Wait for the watch event
	select {
	case <-eventSent:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for watch event")
	}

	// Give handler time to process
	time.Sleep(100 * time.Millisecond)
	close(stopCh)

	added := handler.getAdded()
	assert.Contains(t, added, "new-node")
}

func TestAdapter_ImplementsSharedIndexInformer(t *testing.T) {
	adapter := NewInformerAdapter("/api/v1/nodes", http.DefaultClient, func() runtime.Object {
		return &corev1.Node{}
	})

	// Verify it satisfies the interface at compile time
	var _ cache.SharedIndexInformer = adapter

	// Verify stub methods don't panic
	assert.Nil(t, adapter.GetStore())
	assert.Nil(t, adapter.GetController())
	assert.Nil(t, adapter.GetIndexer())
	assert.Equal(t, "", adapter.LastSyncResourceVersion())
	assert.False(t, adapter.IsStopped())
	assert.NoError(t, adapter.AddIndexers(nil))
	assert.NoError(t, adapter.SetTransform(nil))
	assert.NoError(t, adapter.SetWatchErrorHandler(nil))
	assert.NoError(t, adapter.RemoveEventHandler(nil))
}

func TestAdapter_HasSyncedFalseBeforeList(t *testing.T) {
	adapter := NewInformerAdapter("/api/v1/nodes", http.DefaultClient, func() runtime.Object {
		return &corev1.Node{}
	})
	assert.False(t, adapter.HasSynced(), "should not be synced before Run()")
}

func TestAdapter_UnmarshalError(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("watch") != "true" {
			fmt.Fprint(w, `{"items":[]}`)
			return
		}
		flusher, _ := w.(http.Flusher)
		// Send an event with invalid object data for Node type
		fmt.Fprintln(w, `{"type":"ADDED","object":{"this":"is not a node","metadata":{"name":"bad"}}}`)
		// Send a valid event
		event := fmt.Sprintf(`{"type":"ADDED","object":%s}`, newNodeJSON("good-node"))
		fmt.Fprintln(w, event)
		flusher.Flush()
		<-r.Context().Done()
	}))
	defer server.Close()

	adapter := NewInformerAdapterForTest(server.URL, "/api/v1/nodes", server.Client(), func() runtime.Object {
		return &corev1.Node{}
	})

	handler := &testHandler{}
	adapter.AddEventHandler(handler)

	stopCh := make(chan struct{})
	go adapter.Run(stopCh)

	// Wait for events to process
	time.Sleep(2 * time.Second)
	close(stopCh)

	// Both events should be delivered — JSON unmarshal into Node is lenient
	// (unknown fields are ignored), so "bad" node will still parse with name="bad"
	added := handler.getAdded()
	assert.Contains(t, added, "good-node")
}

func TestAdapter_MultipleHandlers(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("watch") != "true" {
			fmt.Fprintf(w, `{"items":[%s]}`, newNodeJSON("shared-node"))
			return
		}
		<-r.Context().Done()
	}))
	defer server.Close()

	adapter := NewInformerAdapterForTest(server.URL, "/api/v1/nodes", server.Client(), func() runtime.Object {
		return &corev1.Node{}
	})

	handler1 := &testHandler{}
	handler2 := &testHandler{}
	adapter.AddEventHandler(handler1)
	adapter.AddEventHandler(handler2)

	stopCh := make(chan struct{})
	go adapter.Run(stopCh)

	require.Eventually(t, func() bool {
		return adapter.HasSynced()
	}, 5*time.Second, 100*time.Millisecond)

	close(stopCh)

	// Both handlers should receive the event
	assert.Equal(t, []string{"shared-node"}, handler1.getAdded())
	assert.Equal(t, []string{"shared-node"}, handler2.getAdded())
}

func TestAdapter_DeleteEvent(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("watch") != "true" {
			fmt.Fprint(w, `{"items":[]}`)
			return
		}
		flusher, _ := w.(http.Flusher)
		event := fmt.Sprintf(`{"type":"DELETED","object":%s}`, newNodeJSON("removed-node"))
		fmt.Fprintln(w, event)
		flusher.Flush()
		<-r.Context().Done()
	}))
	defer server.Close()

	adapter := NewInformerAdapterForTest(server.URL, "/api/v1/nodes", server.Client(), func() runtime.Object {
		return &corev1.Node{}
	})

	handler := &testHandler{}
	adapter.AddEventHandler(handler)

	stopCh := make(chan struct{})
	go adapter.Run(stopCh)

	time.Sleep(2 * time.Second)
	close(stopCh)

	assert.Contains(t, handler.getDeleted(), "removed-node")
}

func TestAdapter_ModifiedEventHasNilOldObj(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("watch") != "true" {
			fmt.Fprint(w, `{"items":[]}`)
			return
		}
		flusher, _ := w.(http.Flusher)
		event := fmt.Sprintf(`{"type":"MODIFIED","object":%s}`, newNodeJSON("updated-node"))
		fmt.Fprintln(w, event)
		flusher.Flush()
		<-r.Context().Done()
	}))
	defer server.Close()

	// Custom handler that verifies oldObj is nil
	var oldObjWasNil bool
	var mu sync.Mutex

	adapter := NewInformerAdapterForTest(server.URL, "/api/v1/nodes", server.Client(), func() runtime.Object {
		return &corev1.Node{}
	})

	adapter.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(oldObj, newObj interface{}) {
			mu.Lock()
			defer mu.Unlock()
			oldObjWasNil = (oldObj == nil)
		},
	})

	stopCh := make(chan struct{})
	go adapter.Run(stopCh)

	time.Sleep(2 * time.Second)
	close(stopCh)

	mu.Lock()
	defer mu.Unlock()
	assert.True(t, oldObjWasNil, "oldObj should be nil in MODIFIED events from k8swatch (no cache)")
}
