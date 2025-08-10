# phonehome

Package phonehome provides a client for sending telemetry events ("phone home")
to a Segment-compatible backend, as well as HTTP/gRPC server interceptors and
periodic data gatherers for automatic identity updates.

See [Examples](examples_test.go).

## User Consent

As the user consent might not be known at the moment of the client creation, the
consent state is initially unknown, which blocks all telemetry communication.
To grant or withdraw the consent, the client provides the corresponding methods.
  
If no decision is made the client is disabled after `consentTimeout`.

## Client Identity

Client identity (a map of traits) needs to be computed and sent either before
any Track event, or within the first one. Otherwise, the identity will not be
associated with the event.

If client is configured to wait for the initial identity, all Track calls will
be blocked until it is explicitly allowed via a call to `InitialIdentitySent()`.

## Storage Key

A storage key is required for the client to communicate with the storage
platform. If no key is provided at the moment of the client initialization, a
remote configuration will be fetched from the provided configuration URL. This
configuration includes the key. Until the key is acquired, all telemetry
communication is blocked.

If no key is fetched after `storageKeyTimeout`, the key is left empty, and the
waiting calls to `Telemeter()` will return a no-op telemeter instance.

## Execution Environment

A special care has to be taken to prevent accidental telemetry communication to
the production endpoints.

An excution environment is considered to be _release_, if:

- the binary is compiled with `release` flag and without `test` flag;
- the product version has no `-`.

(See `version.IsReleaseVersion()`.)

### Non-release Environment

For testing purposes, the key has to be communicated at the moment of the client
initialization. Otherwise a no-op client is constructed. Other than that, when a
remote configuration is downloaded, the remote key is discarded.

### Release Environment

If a key is not provided at the moment of the client initialization, but a
configuration URL is provided, a periodic reconfiguration is scheduled with
`reconfigurationPeriod`.

## API Interceptors

A client may construct gRPC and HTTP interceptors, that could be used to
configure gRPC and HTTP servers accordingly.

## Periodic data gatherer

Client identity may be gathered and communicated periodically at the given
period. An Identity event with the gathered traits and a Track event are sent at
every period. The Track event makes the identity effective, and may serve as a
heartbeat.
