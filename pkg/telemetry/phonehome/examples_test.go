//go:build test

package phonehome_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/stackrox/rox/pkg/telemetry/phonehome"
	"github.com/stackrox/rox/pkg/telemetry/phonehome/telemeter"
)

func printMessage(message map[string]any) {
	fmt.Printf("---")
	for _, key := range []string{"type", "event", "traits", "properties", "context"} {
		if message[key] != nil {
			fmt.Printf("  %s: %v\n", key, message[key])
		}
	}
}

func newMockServer() (chan map[string]any, *httptest.Server) {
	data := make(chan map[string]any, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		d := json.NewDecoder(r.Body)
		var message map[string][]map[string]any
		d.Decode(&message)
		for _, m := range message["batch"] {
			data <- m
		}
	}))
	return data, server
}

// ExampleNewClient is an example of a simple client, that only sends a couple
// of events.
func ExampleNewClient() {
	data, server := newMockServer()
	defer close(data)
	defer server.Close()

	c := phonehome.NewClient(
		phonehome.WithClient("example", "Test", "v0.0.1"),
		phonehome.WithConnectionConfiguration(server.URL, "segment-api-key", ""),
		phonehome.WithBatchSize(1),
	)

	// Confirm the user has not opted-out from telemetry collection.
	// Until this is clarified, any attempt to call Telemeter() will block.
	c.GrantConsent()

	t := c.Telemeter()

	// Graceful shutdown flushes the buffer.
	defer t.Stop()

	t.Identify(telemeter.WithTraits(map[string]any{
		"Color": "Orange",
	}))
	printMessage(<-data)

	// This call will add the client to the group.
	t.Group(telemeter.WithGroup("Backend", "X"))
	printMessage(<-data)

	t.Track("backend started", map[string]any{
		"Startup duration seconds": 42,
	})
	printMessage(<-data)

	// Output:
	// ---  type: identify
	//   traits: map[Color:Orange]
	//   context: map[device:map[type:Test Server] traits:map[Color:Orange]]
	// ---  type: group
	//   context: map[device:map[type:Test Server] groups:map[Backend:[X]]]
	// ---  type: track
	//   event: backend started
	//   properties: map[Startup duration seconds:42]
	//   context: map[device:map[type:Test Server]]
}

// ExampleClient_Gatherer shows the use of periodic Gatherer.
// The complexity of this example comes from the need of synchronization: any
// Track event actualizes the client identity at the moment of the event, so
// the said identity needs to be gathered and enqueued before. Otherwise, the
// events will not be identifiable by client properties.
func ExampleClient_Gatherer() {
	data, server := newMockServer()
	defer close(data)
	defer server.Close()

	c := phonehome.NewClient(
		phonehome.WithClient("example", "Test", "v0.0.1"),
		phonehome.WithConnectionConfiguration(server.URL, "segment-api-key", ""),
		phonehome.WithAwaitInitialIdentity(),
		phonehome.WithBatchSize(1),
	)

	// Confirm the user has not opted-out from telemetry collection.
	// Until this is clarified, any attempt to call Telemeter() will block.
	c.GrantConsent()

	// Graceful shutdown flushes the buffer.
	defer c.Telemeter().Stop()

	// Gatherer collects and enqueus *some* client identity.
	// Additional identity can be sent by calling Telemeter().Identify().
	g := c.Gatherer()
	g.AddGatherer(func(context.Context) (map[string]any, error) {
		return map[string]any{
			"Color": "Orange",
		}, nil
	})
	g.Start()
	printMessage(<-data) // Gathered identity.

	c.Identify(telemeter.WithTraits(map[string]any{
		"Shape": "Cube",
	}))
	printMessage(<-data) // Additional identity.

	// This will unblock the "Updated example identity" Track event, sent by
	// the started gatherer.
	c.InitialIdentitySent()
	printMessage(<-data) // "Updated example identity" Track event.

	c.Track("backend started", map[string]any{
		"Startup duration seconds": 42,
	})
	printMessage(<-data) // "backend started" Track event.

	// Output:
	// ---  type: identify
	//   traits: map[Color:Orange]
	//   context: map[device:map[type:Test Server] traits:map[Color:Orange]]
	// ---  type: identify
	//   traits: map[Shape:Cube]
	//   context: map[device:map[type:Test Server] traits:map[Shape:Cube]]
	// ---  type: track
	//   event: Updated Test Identity
	//   context: map[device:map[type:Test Server]]
	// ---  type: track
	//   event: backend started
	//   properties: map[Startup duration seconds:42]
	//   context: map[device:map[type:Test Server]]
}

func ExampleClient_AddInterceptorFuncs() {
	data, server := newMockServer()
	defer close(data)
	defer server.Close()

	c := phonehome.NewClient(
		phonehome.WithClient("example", "test", "v0.0.1"),
		phonehome.WithConnectionConfiguration(server.URL, "segment-api-key", ""),
		phonehome.WithBatchSize(1),
	)
	c.AddInterceptorFuncs("API Call",
		func(rp *phonehome.RequestParams, props map[string]any) bool {
			props["path"] = rp.Path
			props["status"] = rp.Code
			return true
		})
	c.GrantConsent()

	myServiceHandler := http.NotFoundHandler()

	mux := http.NewServeMux()
	mux.Handle("/", c.GetHTTPInterceptor()(myServiceHandler))
	mux.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest(http.MethodGet, "/service", bytes.NewReader([]byte{})),
	)

	printMessage(<-data)
	// Output:
	// ---  type: track
	//   event: API Call
	//   properties: map[path:/service status:404]
}

func ExampleClient_Reconfigure() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{
			"storage_key_v1": "new-key",
			"api_call_campaign": [{"method": "{put,delete}"}]
		}`))
	}))
	defer server.Close()

	c := phonehome.NewClient(
		phonehome.WithClient("example", "test", "v0.0.1"),
		phonehome.WithConnectionConfiguration(server.URL, "old-key", server.URL),
		phonehome.WithAwaitInitialIdentity(),
		phonehome.WithBatchSize(1),
		phonehome.WithConfigureCallback(func(rc *phonehome.RuntimeConfig) {
			s, _ := json.Marshal(rc)
			fmt.Println(string(s))
		}),
	)
	// Reconfigure will fetch the configuration from the provided ConfigURL.
	// This will happen automatically in a release environment if no storage key
	// is provided on the client creation.
	// In non-release environments the remote key value will be ignored, but
	// the API call campaign left as is. In such environments an initial value
	// has to be provided via the client configuration.
	c.Reconfigure()
	fmt.Println("Effective storage key:", c.GetStorageKey())
	// Output:
	// {"storage_key_v1":"old-key","api_call_campaign":[{"method":"{put,delete}"}]}
	// Effective storage key: old-key
}
