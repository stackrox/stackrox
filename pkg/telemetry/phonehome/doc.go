/*
Package phonehome provides interfaces and implementation for telemetry data
collection.

The package provides the following entities:

  - client configuration;
  - Telemeter interface, and its [segment] implementation;
  - API interceptors;
  - periodic data gatherer.

# Components

## Client configuration

Client configuration consists of telemetry configuration such as the telemetry
service key, data gathering period, etc. This configuration is used for creating
instances of the Telemeter (Segment client), the API interceptors and the
periodic gatherer.

## Telemeter interface

The [Telemeter] interface allows for sending messages to the configured service
via the following methods:

  - [Telemeter.Track] for live events, which describe something that has
    happened after a user action or triggered by other means;
  - [Telemeter.Identify] for reporting client properties, which represent
    some client traits;
  - [Telemeter.Group] for adding the client to some group and providing the
    group related traits, if any;
  - [Telemeter.Stop] gracefully shutdowns the implementation, which may flush
    the collected messages.

## API Interceptors

API interceptors (gRPC and HTTP) created from a client configuration, when added
to the list of server interceptors and allow for injecting custom events based
on the intercepted request parameters (see [Config.AddInterceptorFunc]). The
list of functions, associated with an event are executed in the order of
addition. If any of the functions associated with an event returns false, the
event won't be tracked. The first function in the chain of every event may serve
as a filter which stops the chain if the intercepted request does not belong to
the event.

The collected events will be tracked by the client telemeter instance.

## Periodic data gatherer

The gatherer created from a client configuration (see [Gatherer.AddGatherer])
allows for adding custom functions which will be executed at the specified time
period. They're supposed to collect some client traits and return as a map of
properties, which will in turn be reported as the client identity by the client
telemeter instance.

# Basic Usage Example

This example shows how to track an event.

	import (
		"github.com/stackrox/rox/pkg/telemetry/phonehome"
	)
	...
	// Instantiate a client configuration:
	cfg := &phonehome.ClientConfig{
		ClientID:   "client id",
		ClientName: "backend",
		StorageKey: "segment-api-key",
	}
	...
	if cfg.Enabled() {
		cfg.Telemeter().Track("backend started", cfg.ClientID, nil)
	}
	...
	// Graceful telemeter shutdown with a buffer flush:
	cfg.Telemeter().Stop()

# Advanced Usage Example

This example shows how to track an event, add a gRPC server interceptor to issue
request related events, and add a data gathering procedure for periodic client
identity update.

	import (
		"github.com/stackrox/rox/pkg/telemetry/phonehome"
		"google.golang.org/grpc"
	)
	...
	// Instantiate a client configuration:
	cfg := &phonehome.ClientConfig{
		ClientID:   "client id",
		ClientName: "backend",
		StorageKey: "segment-api-key",
	}
	...
	if cfg.Enabled() {
		cfg.Telemeter().Track("event", cfg.ClientID, nil)
		cfg.Gatherer().AddGatherer(func(ctx context.Context) (map[string]any, error) {
			return map[string]any{
				// Backend identity properties:
				"version": "backend version",
			}, nil
		})
		// Start periodic gathering:
		cfg.Gatherer().Start()
		// Add an event handler to the interceptor
		cfg.AddInterceptorFunc("gRPC call", func (rp *phonehome.RequestParams, props map[string]any) bool {
			// Filter requests that cause this event:
			if rp.Code == 0 {
				return false
			}
			// Add event related properties:
			props["Code"] = rp.Code
			props["Method"] = rp.Method
			return true
		})
		// Add the interceptor to your server:
		grpc.UnaryInterceptor(cfg.GetGRPCInterceptor())
	}
	...
	// Stop periodic gathering:
	cfg.Gatherer().Stop()
	// Graceful telemeter shutdown with a buffer flush:
	cfg.Telemeter().Stop()

[segment]: https://segment.com
*/
package phonehome
