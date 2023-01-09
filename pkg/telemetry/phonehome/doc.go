/*
Package phonehome provides interfaces and implementation for telemetry data
collection.

The package provides the following entities:

  - client configuration;
  - Telemeter interface, and its [segment] implementation;
  - API interceptors;
  - periodic data gatherer.

# Client configuration

Client configuration consists of telemetry configuration such as the telemetry
service key, data gathering period, etc. This configuration is used for creating
instances of the Telemeter (Segment client), the API interceptors and the
periodic gatherer.

# Telemeter interface

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

# API Interceptors

API interceptors (gRPC and HTTP) created from a client configuration, when added
to the list of server interceptors and allow for injecting custom events based
on the intercepted request parameters (see [Config.AddInterceptorFunc]). The
list of functions, associated with an event are executed in the order of
addition. If any of the functions associated with an event returns false, the
event won't be tracked. The first function in the chain of every event may serve
as a filter which stops the chain if the intercepted request does not belong to
the event.

The collected events will be tracked by the client telemeter instance.

# Periodic data gatherer

The gatherer created from a client configuration (see [Gatherer.AddGatherer])
allows for adding custom functions which will be executed at the specified time
period. They're supposed to collect some client traits and return as a map of
properties, which will in turn be reported as the client identity by the client
telemeter instance.

[segment]: https://segment.com
*/
package phonehome
