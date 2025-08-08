package phonehome_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/stackrox/rox/pkg/eventual"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
	"github.com/stackrox/rox/pkg/telemetry/phonehome/telemeter"
)

func printMessage(message map[string]any) {
	fmt.Printf("---")
	for _, key := range []string{"type", "event", "traits", "properties"} {
		if message[key] != nil {
			fmt.Printf("  %s: %v\n", key, message[key])
		}
	}
}

func newMockServer() (chan map[string]any, *httptest.Server) {
	data := make(chan map[string]any)
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

	c := phonehome.NewClient(&phonehome.Config{
		ClientID:   "username",
		ClientName: "example",
		Endpoint:   server.URL,
		BatchSize:  1,
		StorageKey: eventual.Now("segment-api-key"),
	})

	// Confirm the user has not opted-out from telemetry collection.
	// Until this is clarified, any attempt to call Telemeter() will block.
	c.Enable()

	t := c.Telemeter()

	// Graceful shutdown flushes the buffer.
	defer t.Stop()

	go t.Identify(telemeter.WithTraits(map[string]any{
		"Color": "Orange",
	}))

	printMessage(<-data)

	go t.Track("backend started", map[string]any{
		"Startup duration seconds": 42,
	})

	printMessage(<-data)

	// Output:
	// ---  type: identify
	//   traits: map[Color:Orange]
	// ---  type: track
	//   event: backend started
	//   properties: map[Startup duration seconds:42]
}

// ExampleGatherer shows the use of periodic Gatherer.
// The complexity of this example comes from the need of synchronization: any
// Track event actualizes the client identity at the moment of the event, so
// the said identity needs to be gathered and enqueued before. Otherwise, the
// events will not be identifiable by client properties.
func ExampleGatherer() {
	data, server := newMockServer()
	defer close(data)
	defer server.Close()

	c := phonehome.NewClient(&phonehome.Config{
		ClientID:   "username",
		ClientName: "example",
		Endpoint:   server.URL,
		BatchSize:  1,
		StorageKey: eventual.Now("segment-api-key"),
		// Identified will be set by the gatherer after the first identity is
		// sent. This will unblock potentially waiting Track events.
		Identified: eventual.New[bool](),
	})

	// Confirm the user has not opted-out from telemetry collection.
	// Until this is clarified, any attempt to call Telemeter() will block.
	c.Enable()

	// Graceful shutdown flushes the buffer.
	defer c.Telemeter().Stop()

	// Gatherer collects and enqueus the client identity and unblocks
	// potentially waiting Track calls.
	g := c.Gatherer()
	g.AddGatherer(func(context.Context) (map[string]any, error) {
		return map[string]any{
			"Color": "Orange",
		}, nil
	})
	g.Start()

	printMessage(<-data)
	printMessage(<-data)

	go func() {
		// This Track call is synchronous with the output data channel: as the
		// batch size is set to 1, we'll ensure a message is sent before sending
		// another one.
		c.Telemeter().Track("backend started", map[string]any{
			"Startup duration seconds": 42,
		})
	}()

	printMessage(<-data)

	// Output:
	// ---  type: identify
	//   traits: map[Color:Orange]
	// ---  type: track
	//   event: Updated example Identity
	// ---  type: track
	//   event: backend started
	//   properties: map[Startup duration seconds:42]
}
