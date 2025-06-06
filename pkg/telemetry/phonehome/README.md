# phonehome

`phonehome` is a Go library that makes it easy to collect and send telemetry to
any Segment-compatible endpoint. It includes:

- a configurable telemetry client (with consent, identity and storage-key
  management);
- HTTP & gRPC interceptors;
- optional periodic identity/heartbeat reporting.

See [Examples](examples_test.go).

## User Consent

User consent is unknown at client creation, so telemetry is blocked until you
call GrantConsent() or WithdrawConsent(). If no decision is made within
`consentTimeout`, telemetry is automatically disabled.

## Client Identity and Groups

- Client identity (map of traits) must be sent before or within the first Track
  event—otherwise traits won’t be attached. The same applies to groups.
- If you enable `WithAwaitInitialIdentity()` option, all Track calls block until
  you call `InitialIdentitySent()`.

## Storage Key

A storage key is required for the client to communicate with the storage
platform. If no key is provided at the moment of the client initialization, a
remote configuration will be fetched from the provided configuration URL. This
configuration includes the key. Until the key is acquired, all telemetry
communication is blocked.

If no key is fetched after `storageKeyTimeout`, the key is left empty, and the
waiting calls to `Telemeter()` will return a no-op telemeter instance.

## Build & Environment Modes

Take care to avoid sending telemetry to production by mistake.

An execution environment is considered to be a _release_, if the binary:

- is built with `release` flag and without `test` flag;
- has version string contains no hyphens.

(See `version.IsReleaseVersion()`.)

### Non-release Environment

For testing purposes, the key has to be communicated at the moment of the client
initialization. Otherwise a no-op client is constructed. Other than that, when a
remote configuration is downloaded, the remote key is discarded.

### Release Environment

If a key is not provided at the moment of the client initialization, but a
configuration URL is provided, a periodic reconfiguration is scheduled with
`reconfigurationPeriod`.

### CI Environment

If a release version is being tested in a CI environment, the storage key has to
be explicitly set to "DISABLED".

### Staging Environment

To enable telemetry reporting in a staging environment, set the storage key to
the staging value.

## API Interceptors

A client may construct gRPC and HTTP interceptors, that could be used to
configure gRPC and HTTP servers accordingly.

## Periodic Identity & Heartbeat

Client identity may be gathered and communicated periodically at the given
period. An Identity event with the gathered traits and a Track event are sent at
every period. The Track event makes the identity effective, and may serve as a
heartbeat.

## Message Deduplication

This package provides the `WithNoDuplicates` option to generate message
identifiers in a form of `<prefix>-<hash>`, where `prefix` is provided as the
option argument, and `hash` is a hash of message content.

If the `WithNoDuplicates` option is supplied, the built-in expiring cache will
identify duplicate messages if they appear during 24h window.

The prefix can be computed dynamically based on current time to allow duplicates
after some specific window.

A similar mechanism is implemented on the Segment server side:

> Segment guarantees that 99% of your data won’t have duplicates within an
> approximately 24 hour look-back window. Warehouses and Data Lakes also have
> their own secondary deduplication process to ensure you store clean data.
>
> ...
>
> Segment deduplicates on the event’s messageId, not on the contents of the
> event payload.

_Source: [segment.com](https://segment.com/docs/guides/duplicate-data/)._
