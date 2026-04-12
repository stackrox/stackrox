package k8swatch

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestEvent creates a watch event JSON line for testing.
func newTestEvent(eventType, name, namespace, rv string) string {
	return fmt.Sprintf(`{"type":%q,"object":{"metadata":{"name":%q,"namespace":%q,"resourceVersion":%q}}}`,
		eventType, name, namespace, rv)
}

func TestWatcher_ReceivesEvents(t *testing.T) {
	var received []string
	var mu sync.Mutex

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.RawQuery, "watch=true")
		flusher, ok := w.(http.Flusher)
		require.True(t, ok)

		for _, event := range []string{
			newTestEvent("ADDED", "pod-1", "default", "100"),
			newTestEvent("MODIFIED", "pod-1", "default", "101"),
			newTestEvent("DELETED", "pod-1", "default", "102"),
		} {
			fmt.Fprintln(w, event)
			flusher.Flush()
		}
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	w := &Watcher{
		baseURL: server.URL, apiPath: "/api/v1/pods",
		client:  server.Client(),
		handler: func(eventType string, raw json.RawMessage) {
			mu.Lock()
			defer mu.Unlock()
			received = append(received, eventType)
		},
	}

	// Override the URL to point to our test server
	err := w.doWatch(ctx, "", func(event Event, rv string) {
		if event.Type != "BOOKMARK" {
			w.handler(event.Type, event.Object)
		}
	})
	// Server closes connection after sending events — scanner returns nil
	assert.NoError(t, err)

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, []string{"ADDED", "MODIFIED", "DELETED"}, received)
}

func TestWatcher_BookmarkUpdatesResourceVersion(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		flusher, _ := w.(http.Flusher)
		// Send a bookmark event
		fmt.Fprintln(w, newTestEvent("BOOKMARK", "", "", "500"))
		flusher.Flush()
		// Then a real event
		fmt.Fprintln(w, newTestEvent("ADDED", "svc-1", "kube-system", "501"))
		flusher.Flush()
	}))
	defer server.Close()

	var handlerCalled atomic.Int32

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	w := &Watcher{
		baseURL: server.URL, apiPath: "/api/v1/services",
		client:  server.Client(),
		handler: func(eventType string, raw json.RawMessage) {
			handlerCalled.Add(1)
			// Bookmark should NOT reach the handler
			assert.NotEqual(t, "BOOKMARK", eventType)
		},
	}

	var lastRV string
	w.doWatch(ctx, "", func(event Event, rv string) {
		lastRV = rv
		if event.Type != "BOOKMARK" {
			w.handler(event.Type, event.Object)
		}
	})

	assert.Equal(t, "501", lastRV)
	assert.Equal(t, int32(1), handlerCalled.Load(), "handler should be called once (ADDED only, not BOOKMARK)")
}

func TestWatcher_410GoneTriggersRelist(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// If resourceVersion is set, return 410 Gone
		if r.URL.Query().Get("resourceVersion") != "" {
			w.WriteHeader(http.StatusGone)
			fmt.Fprintln(w, `{"kind":"Status","code":410,"reason":"Gone","message":"too old"}`)
			return
		}
		flusher, _ := w.(http.Flusher)
		fmt.Fprintln(w, newTestEvent("ADDED", "node-1", "", "1"))
		flusher.Flush()
	}))
	defer server.Close()

	w := &Watcher{
		baseURL: server.URL, apiPath: "/api/v1/nodes",
		client:  server.Client(),
		handler: func(eventType string, raw json.RawMessage) {},
	}

	// First call with resourceVersion — should get 410
	err := w.doWatch(context.Background(), "old-version", func(event Event, rv string) {})
	assert.True(t, isGone(err), "expected 410 Gone error")

	// Second call without resourceVersion — should succeed
	err = w.doWatch(context.Background(), "", func(event Event, rv string) {})
	assert.NoError(t, err)
}

func TestWatcher_ReconnectsOnError(t *testing.T) {
	var connectCount atomic.Int32

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := connectCount.Add(1)
		if count <= 2 {
			// First two connections: close immediately (simulates network error)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		// Third connection: send an event then close
		flusher, _ := w.(http.Flusher)
		fmt.Fprintln(w, newTestEvent("ADDED", "deploy-1", "default", "1"))
		flusher.Flush()
	}))
	defer server.Close()

	var eventReceived atomic.Bool
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	w := &Watcher{
		baseURL: server.URL, apiPath: "/apis/apps/v1/deployments",
		client:  server.Client(),
		handler: func(eventType string, raw json.RawMessage) {
			eventReceived.Store(true)
			cancel() // Stop after receiving the event
		},
	}

	w.Run(ctx)

	assert.True(t, eventReceived.Load(), "should have received event after reconnection")
	assert.GreaterOrEqual(t, connectCount.Load(), int32(3), "should have reconnected at least twice")
}

func TestWatcher_MalformedJSONSkipped(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		flusher, _ := w.(http.Flusher)
		fmt.Fprintln(w, `{not valid json}`)
		fmt.Fprintln(w, newTestEvent("ADDED", "valid-pod", "default", "1"))
		flusher.Flush()
	}))
	defer server.Close()

	var received []string
	var mu sync.Mutex

	w := &Watcher{
		baseURL: server.URL, apiPath: "/api/v1/pods",
		client:  server.Client(),
		handler: func(eventType string, raw json.RawMessage) {
			mu.Lock()
			defer mu.Unlock()
			received = append(received, eventType)
		},
	}

	w.doWatch(context.Background(), "", func(event Event, rv string) {
		if event.Type != "BOOKMARK" {
			w.handler(event.Type, event.Object)
		}
	})

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, []string{"ADDED"}, received, "should skip malformed JSON and process valid event")
}

func TestIsGone(t *testing.T) {
	assert.True(t, isGone(&goneError{}))
	assert.False(t, isGone(nil))
	assert.False(t, isGone(fmt.Errorf("some other error")))
}
