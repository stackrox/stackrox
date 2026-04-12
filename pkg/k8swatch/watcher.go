// Package k8swatch provides a minimal Kubernetes resource watcher using raw
// HTTP watch streams. It replaces client-go informers for use cases that
// only need event forwarding (no local cache, no indexing).
//
// Compared to client-go informers:
//   - Zero object cache (events are forwarded immediately)
//   - 1 goroutine per resource type (not 2)
//   - No scheme registration at init() (~10 MB RSS savings)
//   - No dependency on typed client packages
//   - Handles: reconnection, 410 Gone, bookmarks, token rotation, backoff
package k8swatch

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/stackrox/rox/pkg/logging"
)

var log = logging.LoggerForModule()

// Event represents a Kubernetes watch event.
type Event struct {
	Type   string          `json:"type"` // ADDED, MODIFIED, DELETED, BOOKMARK, ERROR
	Object json.RawMessage `json:"object"`
}

// Handler is called for each watch event. The raw JSON object is provided
// for the handler to deserialize as needed (typed or unstructured).
type Handler func(eventType string, object json.RawMessage)

// Watcher watches a Kubernetes API resource and delivers events to a handler.
type Watcher struct {
	apiPath string
	handler Handler
	client  *http.Client
}

// New creates a watcher for the given API path (e.g., "/apis/apps/v1/deployments").
// The client should be configured with TLS for the API server.
func New(apiPath string, client *http.Client, handler Handler) *Watcher {
	return &Watcher{
		apiPath: apiPath,
		handler: handler,
		client:  client,
	}
}

// Run starts watching and blocks until the context is cancelled.
// It automatically reconnects on failures with exponential backoff.
func (w *Watcher) Run(ctx context.Context) {
	var resourceVersion string
	backoff := time.Second

	var eventCount, reconnectCount int
	log.Infof("k8swatch: starting watch for %s", w.apiPath)

	for {
		if ctx.Err() != nil {
			log.Infof("k8swatch: %s stopped (context cancelled), total events: %d, reconnects: %d", w.apiPath, eventCount, reconnectCount)
			return
		}

		reconnectCount++
		log.Infof("k8swatch: %s connecting (attempt %d, resourceVersion=%q)", w.apiPath, reconnectCount, resourceVersion)

		err := w.doWatch(ctx, resourceVersion, func(event Event, rv string) {
			if rv != "" {
				resourceVersion = rv
			}
			if event.Type == "BOOKMARK" {
				log.Debugf("k8swatch: %s bookmark rv=%s", w.apiPath, rv)
				return
			}
			eventCount++
			if eventCount <= 5 || eventCount%100 == 0 {
				log.Infof("k8swatch: %s event #%d type=%s rv=%s", w.apiPath, eventCount, event.Type, rv)
			}
			w.handler(event.Type, event.Object)
			backoff = time.Second
		})

		if ctx.Err() != nil {
			return
		}

		if isGone(err) {
			log.Infof("Watch %s: 410 Gone, restarting from scratch", w.apiPath)
			resourceVersion = ""
			backoff = time.Second
			continue
		}

		if err != nil {
			log.Warnf("Watch %s: %v (reconnecting in %v)", w.apiPath, err, backoff)
		}

		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return
		}
		if backoff < 30*time.Second {
			backoff *= 2
		}
	}
}

func (w *Watcher) doWatch(ctx context.Context, resourceVersion string, handle func(Event, string)) error {
	url := fmt.Sprintf("https://kubernetes.default.svc%s?watch=true&allowWatchBookmarks=true", w.apiPath)
	if resourceVersion != "" {
		url += "&resourceVersion=" + resourceVersion
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	// Re-read token on each connection to handle rotation.
	token, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
	if err != nil {
		return fmt.Errorf("reading service account token: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+string(token))

	resp, err := w.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusGone {
		io.Copy(io.Discard, resp.Body)
		return &goneError{}
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, body)
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 1<<20), 1<<20)
	for scanner.Scan() {
		var event Event
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			log.Warnf("Watch %s: unmarshal error: %v", w.apiPath, err)
			continue
		}

		// Extract resourceVersion from the object metadata.
		var meta struct {
			Metadata struct {
				ResourceVersion string `json:"resourceVersion"`
			} `json:"metadata"`
		}
		json.Unmarshal(event.Object, &meta)

		handle(event, meta.Metadata.ResourceVersion)
	}
	return scanner.Err()
}

type goneError struct{}

func (e *goneError) Error() string { return "410 Gone" }

func isGone(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(*goneError)
	return ok
}
