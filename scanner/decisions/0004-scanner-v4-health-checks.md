# 0004 - Scanner v4 Health Check API

- **Author(s):** Ross Tannenbaum
- **Created:** 2023-08-28

## Status

Accepted.

## Context

The Scanner v4 and ScannerDB v4 deployment configurations will need to define a `readinessProbe` which the kubelet will periodically probe to determine each container's readiness.

## Decision

Scanner v4 Indexer and Matcher container configurations will both define a `grpc` probe hosted at the container's configured gRPC port (exact port number is not addressed here).

The [gRPC specification](https://github.com/grpc/grpc/blob/master/doc/health-checking.md) states the server *should* export the following service:

```
syntax = "proto3";

package grpc.health.v1;

message HealthCheckRequest {
  string service = 1;
}

message HealthCheckResponse {
  enum ServingStatus {
    UNKNOWN = 0;
    SERVING = 1;
    NOT_SERVING = 2;
    SERVICE_UNKNOWN = 3;  // Used only by the Watch method.
  }
  ServingStatus status = 1;
}

service Health {
  rpc Check(HealthCheckRequest) returns (HealthCheckResponse);

  rpc Watch(HealthCheckRequest) returns (stream HealthCheckResponse);
}
```

[Kubernetes specifies](https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#probe-check-methods) the probe is successful if `status` is `SERVING`.

The [Kubernetes gRPC probe implementation](https://github.com/kubernetes/kubernetes/blob/v1.28.1/pkg/probe/grpc/grpc.go) utilizes the [grpc_health_v1](https://pkg.go.dev/google.golang.org/grpc/health/grpc_health_v1) package, which defines the `Health` service in Go. Instead of creating a new protobuf, Scanner v4 will implement the `Health` service as defined in this package.

It is also clear when looking at the implementation that `Watch` is not called. Therefore, we will opt to not implement `Watch` to keep the implementation simple.

Both modes of Scanner v4 execution, Indexer and Matcher, will expose this service (ScannerDB has no need for this). However, both modes will define "ready" (ie `status` of `SERVING`) differently.

The following sections define the necessary conditions for each component to be considered "ready".

### Indexer

The Scanner v4 Indexer will be considered ready simply once the server starts.

### Matcher

The Scanner v4 Matcher will be considered ready once Scanner v4 DB has vulnerabilities populated for each configured vulnerability updater. It will not be required that the vulnerabilities necessarily be up-to-date.

### DB

Scanner v4 DB's `readinessProbe` will match Central DB's. At this time, it is as follows:

```
readinessProbe:
  exec:
    command:
    - /bin/sh
    - -c
    - -e
    - |
      exec pg_isready -U "postgres" -h 127.0.0.1 -p 5432
  failureThreshold: 3
  periodSeconds: 10
  successThreshold: 1
  timeoutSeconds: 1
```

## Consequences

It is possible Scanner v4 Matcher instances return outdated vulnerability results upon startup. We accept this, as the vulnerabilities will eventually (typically within 3 hours if running StackRox in online-mode) be updated, and StackRox Central/Sensor will periodically reprocess image scan results (every 4 hours, by default).

Scanner v4 images are not shipped with vulnerabilities preloaded into Scanner v4 DB at this time, so it is possible the Scanner v4 Matcher may take considerably longer to be "ready" than the current Scanner v2. We accept this now, but we will reassess later as we learn more about production and CI implications of this.
