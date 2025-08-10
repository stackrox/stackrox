# phonehome

Package phonehome provides a client for sending telemetry events ("phone home")
to a Segment-compatible backend, as well as HTTP/gRPC server interceptors and
periodic data gatherers for automatic identity updates.

Please see how telemetry collection should be configured in various environments
[on Confluence].

## Client instantiation

Client configuration consists of telemetry configuration such as the telemetry
service key, data gathering period, etc. This configuration is used for creating
instances of the Telemeter (Segment client), the API interceptors and the
periodic gatherer.

## Telemeter interface

The [telemeter.Telemeter] interface allows for sending messages to the
configured service via the following methods:

- [Telemeter.Track] for live events, which describe something that has
happened after a user action or triggered by other means;
- [Telemeter.Identify] for reporting client or user properties;
- [Telemeter.Group] for adding the client or a user to some group, providing
the group related traits, if any;
- [Telemeter.Stop] gracefully shutdowns the implementation, which may flush
the collected messages.

## Options

- [WithUserID] sets the ID of the user for the call. If not provided,
anonymous ID is set equal to client ID (and device ID);
- [WithClient] overrides the client ID and type to send messages from the name
of another client;
- [WithGroups] adds a list of groups, associated to an event. This may be
helpful in the case when a user or client may belong to several groups, and
some particular event concerns only some of these groups. Amplitude will
inject according groups properties to the events.

## API Interceptors

API interceptors (gRPC and HTTP) created from a client configuration, when added
to the list of server interceptors and allow for injecting custom events based
on the intercepted request parameters (see [Config.AddInterceptorFuncs]). The
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

[Examples] include cases for:

- Basic Usage:
  Create and configure a client, enable it (grant consent), then send identify
  and track calls. The telemeter must be stopped to flush buffered events.
- Periodic Data Gathering
  Use a Gatherer to collect and report client identity or other traits on a schedule.
- HTTP Interceptor
  Instrument your servers to emit events automatically on every request.
- Remote Configuration
  Optionally, download a remote key or campaign settings and reconfigure at runtime.

[Examples]: examples_test.go
[segment]: https://segment.com
[on Confluence]: https://spaces.redhat.com/display/StackRox/Telemetry+Configuration+in+Environments
