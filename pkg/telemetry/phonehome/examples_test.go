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
	fmt.Printf("%s:\n  event name: %v\n  client traits: %v\n  event properties: %v\n",
		message["type"],
		message["event"],
		message["traits"],
		message["properties"])
}

// ExampleNewClient is an example of a simple client, that only sends a couple
// of events.
func ExampleNewClient() {
	data := make(chan map[string]any)
	defer close(data)

	// Start mock telemetry HTTP server.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		d := json.NewDecoder(r.Body)
		var message map[string][]map[string]any
		d.Decode(&message)
		for _, m := range message["batch"] {
			data <- m
		}
	}))
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

	// Graceful shutdown flushes the buffer.
	defer c.Telemeter().Stop()

	// Optionally send client identity information
	c.Telemeter().Identify(nil, telemeter.WithTraits(map[string]any{
		"Color": "Orange",
	}))

	printMessage(<-data)

	c.Telemeter().Track("backend started", map[string]any{
		"Startup duration seconds": 42,
	})

	printMessage(<-data)

	// Output:
	// identify:
	//   event name: <nil>
	//   client traits: map[Color:Orange]
	//   event properties: <nil>
	// track:
	//   event name: backend started
	//   client traits: <nil>
	//   event properties: map[Startup duration seconds:42]
}

// ExampleGatherer shows the use of periodic Gatherer.
// The complexity of this example comes from the need of synchronization: any
// Track event actualizes the client identity at the moment of the event, so
// the said identity needs to be gathered and enqueued before. Otherwise, the
// events will not be identifiable by client properties.
func ExampleGatherer() {
	data := make(chan map[string]any)
	defer close(data)

	// Start mock telemetry HTTP server.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		d := json.NewDecoder(r.Body)
		var message map[string][]map[string]any
		d.Decode(&message)
		for _, m := range message["batch"] {
			data <- m
		}
	}))
	defer server.Close()

	c := phonehome.NewClient(&phonehome.Config{
		ClientID:   "username",
		ClientName: "example",
		Endpoint:   server.URL,
		BatchSize:  1,
		StorageKey: eventual.Now("segment-api-key"),
		Identified: eventual.New[bool](),
	})

	go func() {
		// This will be blocked until the client is enabled.
		c.Telemeter().Track("backend started", map[string]any{
			"Startup duration seconds": 42,
		})
	}()

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
	printMessage(<-data)

	// Output:
	// identify:
	//   event name: <nil>
	//   client traits: map[Color:Orange]
	//   event properties: <nil>
	// track:
	//   event name: Updated example Identity
	//   client traits: <nil>
	//   event properties: <nil>
	// track:
	//   event name: backend started
	//   client traits: <nil>
	//   event properties: map[Startup duration seconds:42]
}
