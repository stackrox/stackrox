# 0001 - ScannerV4 APIs definition

- **Status:** Accepted
- **Created:** [2023-05-09 Tue]

## Context

We are transitioning Stackrox Scanner to use [ClairCore](https://github.com/quay/claircore) as its underlying scanning engine. The goal is to align vulnerability scanning results between Quay/Clair and Stackrox. The transition will be gradual, starting with container image scanning. During this period, both the existing Scanner (ScannerV2) and the new ClairCore-based Scanner (ScannerV4) will be simultaneously deployed.

The move to ClairCore and the adoption of ScannerV4 brings the need for new API definitions. Also, ClairCore allows for Clair deployment modes that better supports the secured cluster use-case (local scanning), horizontal scalability, and high-availability scenarios. The new APIs ideally should align with the ClairCore's capabilities while minimizing changes in Central.

## Decision

ScannerV4 APIs will exclusively use gRPC. ScannerV4 APIs are not backward compatible with ScannerV2.

ScannerV4 will offer modes of operation [akin to Clair's deployment models](https://quay.github.io/clair/howto/deployment.html). Each mode will implement different gRPC services, reflecting the underlying separation between ClairCore's [libindex](https://pkg.go.dev/github.com/quay/claircore/libindex#Libindex) and [libvuln](https://pkg.go.dev/github.com/quay/claircore/libvuln#Libvuln). They will be named "indexer" and "matcher". Both modes can be enabled concurrently.

ScannerV4 will use "index reports" and "vulnerability reports" data models, similar to ClairCore's. Both types will link to a "scannable resource" (e.g., a container image) using a "hash_id", managed by the clients, that uniquely identifies the resource's manifest.

ScannerV4 will replace ScannerV2 as the container image scanner in Central. A new image integration for ScannerV4 will be created and replace ScannerV2's integration (type "clarify"), activated by the feature flag: `ROX_SCANNER_V4_ENABLED`. The "clarify" type will continue to be used for orchestrator and node scanning.

ScannerV4 gRPC endpoints will use GRPC status codes to communicate return status in case of non-successful responses.

ScannerV4 will also provide endpoints for health and observability checks, which are not detailed in this ADR.

### ScannerV4 API Endpoints

The endpoints are versioned at `v4` to align with Clair:

1. `scanner.v4.Indexer/CreateIndex`: Create a manifest of resource and create an index, or re-index. Idempotent while creation is being executed. Synchronous and tied to the client request. Returns [`IndexReport`](https://github.com/quay/claircore/blob/v1.4.18/indexreport.go#L19)
2. `scanner.v4.Indexer/GetIndex`: Retrieve or check index existence. Returns [`IndexReport`](https://github.com/quay/claircore/blob/v1.4.18/indexreport.go#L19).
3. `scanner.v4.Matcher/GetVulnerabilities`: Get vulnerabilities for a given resouce's manifest. Returns [`VulnerabilityReport`](https://github.com/quay/claircore/blob/v1.4.18/vulnerabilityreport.go#L7).
4. `scanner.v4.Matcher/GetVulnerabilityMetadata`: Get information on vulnerability metadata, e.g. last update timestamp.

Example of manifest's `hash_id` usage, to create index reports for container images:

```
message ContainerImageLocator {
    string url      = 1;
    string username = 2;
    string password = 3;
}

message CreateIndexRequest {
    string hash_id = 1;
    oneof resource_locator {
        ContainerImageLocator container_image;
    }
}
```

Example of index request and vulnerability request:

```
message GetIndexRequest {
    string hash_id   = 1;
    // If true, does not return the Index Report, only status.
    bool check_only  = 2;
}

message GetVulnerabilitiesRequest {
    string hash_id = 1;
}
```

Example of common error codes and scenarios:

| Code | Scenario |
| --- | ---
| `UNAVAILABLE` | Scanner temporary or retriable issue caused by dependencies (e.g registries, databases).
| `INVALID_ARGUMENT` | Invalid input or malformed input (e.g. bad credentials, malformed `hash_id`).
| `INTERNAL` | Unrecoverable errors, unexpected condition or invariant (e.g. bugs, broken schema, etc).
| `FAILED_PRECONDITION	` | Index report does not exist for this given `hash_id` (e.g. when calling `GetVulnerabilities`).

All APIs are idempotent, hence retriable on timeouts and temporary errors.

## Consequences

Leveraging gRPC-only helps maintaining service contracts, and enhances performance. It also align with other Stackrox services.

The new image integration for ScannerV4 ensures minimal changes are necessary in Central to switch between ScannerV2 and ScannerV4 using a feature flag.

The Indexer and Matcher modes allow additional deployment options, enabling ScannerV4 to scale and serve different scenarios. The multiple modes and gRPC services pave the way for advanced deployments, such as multi-tenant scanning and node scanning.

Using gRPC exclusively increases complexity for load balancing ScannerV4 for horizontal scalability. Additional steps will be necessary to achieve this, including service-mesh tooling or client-side load-balancing, which are not available out-of-the-box and require additional work.

The client-managed `hash_id` approach for identifying manifests may lead to duplicated index reports for locators pointing to the same resource. While this simplifies the API to support future resource types (orchestrator, nodes), it may lead to client misuse.

## References

1. gRPC Load Balancing on Kubernetes without Tears: https://kubernetes.io/blog/2018/11/07/grpc-load-balancing-on-kubernetes-without-tears/
