# phonehome

Package phonehome provides a client for sending telemetry events ("phone home")
to a Segment-compatible backend, as well as HTTP/gRPC server interceptors and
periodic data gatherers for automatic identity updates.

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
