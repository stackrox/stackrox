# 0001 - ScannerV4 APIs definition

- **Status:** Updated by [#0002](0002-local-scanning-with-scanner-v4.md)
- **Created:** [2023-05-09 Tue]

## Context

We are transitioning StackRox Scanner to use [ClairCore](https://github.com/quay/claircore) as its underlying scanning engine. The goal is to align vulnerability scanning results between Quay/Clair and StackRox. The transition will be gradual, starting with container image scanning. During this period, the existing Scanner (ScannerV2) and the new ClairCore-based Scanner (ScannerV4) will be simultaneously deployed.

The move to ClairCore and the adoption of ScannerV4 bring the need for new API definitions. Also, ClairCore allows for Clair deployment modes that better support the secured cluster use-case (local scanning), horizontal scalability, and high-availability scenarios. The new APIs ideally should align with ClairCore's capabilities while minimizing changes in Central.

## Decision

ScannerV4 APIs will exclusively use gRPC. ScannerV4 APIs are not backward compatible with ScannerV2.

ScannerV4 will offer modes of operation [akin to Clair's deployment models](https://quay.github.io/clair/howto/deployment.html). Each mode will implement different gRPC services, reflecting the underlying separation between ClairCore's [libindex](https://pkg.go.dev/github.com/quay/claircore/libindex#Libindex) and [libvuln](https://pkg.go.dev/github.com/quay/claircore/libvuln#Libvuln). They will be named "indexer" and "matcher". Both modes can be enabled concurrently.

ScannerV4 will use "index reports" and "vulnerability reports" data models, similar to ClairCore's. Both types will link to a "scannable resource" (e.g., a container image) using an ID created by Scanner clients (i.e. `hash_id`). This is a string that uniquely identifies the resource's manifest. To avoid conflicts between different resources, and allow changes to how the IDs are generated, Central and Sensor (the existing clients) will namespace them with `/v1/<resource>/`.  Initially, `/v1/containerimage/<image-digest>` will be supported for container images.

Index Reports are persisted, and their lifecycle is managed by Scanner. Central and Sensor create reports on demand. Scanner is responsible for deleting least-recently used reports.

ScannerV4 gRPC endpoints will use gRPC status codes to communicate the return status in case of non-successful responses.

ScannerV4 will also provide endpoints for health and observability checks, which are not detailed in this ADR.

In Central, ScannerV4 will replace ScannerV2 as the container image scanner. A new image integration for ScannerV4 will be created and replace ScannerV2's integration (type "clarify"), activated by a feature flag. The "clarify" type will continue to be used for orchestrator and node scanning.

### ScannerV4 API Endpoints

The endpoints are versioned at `v4` to align with Clair:

1. `scanner.v4.Indexer/CreateIndexReport`: Create a manifest of a specified resource and create an index or re-index. Idempotent while creation is being executed. Synchronous and tied to the client's request. Returns [`IndexReport`](https://github.com/quay/claircore/blob/v1.4.18/indexreport.go#L19)
2. `scanner.v4.Indexer/GetIndexReport`: Retrieve index report. Returns [`IndexReport`](https://github.com/quay/claircore/blob/v1.4.18/indexreport.go#L19).
3. `scanner.v4.Indexer/HasIndexReport`: Check if index report exists. Returns nothing.
4. `scanner.v4.Matcher/GetVulnerabilities`: Get vulnerabilities for a given resource's manifest. Returns [`VulnerabilityReport`](https://github.com/quay/claircore/blob/v1.4.18/vulnerabilityreport.go#L7).
5. `scanner.v4.Matcher/GetMetadata`: Get information on vulnerability metadata, e.g., last update timestamp.

Example of manifest's `hash_id` usage to create index reports for container images:

```
message ContainerImageLocator {
    string url      = 1;
    string username = 2;
    string password = 3;
}

message CreateIndexReportRequest {
    string hash_id = 1;
    oneof resource_locator {
        ContainerImageLocator container_image;
    }
}
```

Example of index request and vulnerability request:

```
message GetIndexReportRequest {
    string hash_id   = 1;
}

message HasIndexReportRequest {
    string hash_id   = 1;
}

message GetVulnerabilitiesRequest {
    string hash_id = 1;
}
```

Examples of common error codes and scenarios:

| Code | Scenario |
| --- | ---
| `UNAVAILABLE` | Scanner temporary or retriable issue caused by dependencies (e.g., registries, databases).
| `INVALID_ARGUMENT` | Invalid input or malformed input (e.g., bad credentials, malformed `hash_id`).
| `INTERNAL` | Unrecoverable errors, unexpected conditions, or invariant (e.g., bugs, broken schema, etc).
| `FAILED_PRECONDITION	` | Index report does not exist for this given `hash_id` (e.g. when calling `GetVulnerabilities`).

All APIs are idempotent, hence retriable on timeouts and temporary errors.

## Consequences

Leveraging gRPC-only helps maintain service contracts and enhances performance. It also aligns with other StackRox services. Scanner gRPC service will configured using [=stackrox/pkg/grpc=](https://github.com/stackrox/stackrox/blob/74476b76b39dfe2e9cdaeecc3e9eaf262097389f/pkg/grpc), which offers certificate management, default service configuration (e.g., max payload sizer, timeouts), and metrics.

The new image integration for ScannerV4 ensures minimal changes are necessary for Central to switch between ScannerV2 and ScannerV4 using a feature flag.

The Indexer and Matcher modes allow additional deployment options, enabling ScannerV4 to scale and serve different scenarios. The multiple modes and gRPC services pave the way for advanced deployments, such as multi-tenant scanning and node scanning.

Using gRPC exclusively increases complexity for load balancing ScannerV4 for horizontal scalability. Additional steps will be necessary to achieve this, including service-mesh tooling or client-side load-balancing, which are not available out of the box and require additional work.

The client-managed `hash_id` approach for identifying manifests may lead to duplicated index reports for locators pointing to the same resource. While this simplifies the API to support future resource types (orchestrator, nodes), it may lead to client misuse.

The use of a version prefix in the `hash_id` opens the door to gracefully modify the ID format without breaking existing IDs, allowing Scanner to parse them, if needed.

Always re-indexing upon `Indexer/CreateIndexReport` calls is sub-optimal. There are [interfaces in ClairCore's Indexer`](https://github.com/quay/clair/blob/8174e950186c03bee10a9174643bca0f173710c2/indexer/service.go#L47) that allows check to not trigger re-indexing. This could potentially be leveraged by ScannerV4 to optimize re-indexing, a transparent change for customers.

## References

1. gRPC Load Balancing on Kubernetes without Tears: https://kubernetes.io/blog/2018/11/07/grpc-load-balancing-on-kubernetes-without-tears/
