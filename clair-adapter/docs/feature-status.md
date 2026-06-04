# Clair Adapter: Feature Status & Scanner V4 Gap Analysis

## What the Clair Adapter Is

The clair-adapter is a Go service that replaces Scanner V4 by delegating container image indexing and vulnerability matching to upstream Clair while preserving the Scanner V4 gRPC API. Central and Sensor connect to it using the same protos — it's a drop-in replacement.

Unlike Scanner V4 (which embeds ClairCore as a Go library), the adapter calls Clair over HTTP and applies StackRox-specific enrichments in-process.

## Current Features

### gRPC API (6 of 10 RPCs implemented)

| RPC                         | Status          | Notes                                                  |
|-----------------------------|-----------------|--------------------------------------------------------|
| `CreateIndexReport`         | Implemented     | Fetches image manifest from registry, submits to Clair |
| `GetIndexReport`            | Implemented     | Retrieves from Clair                                   |
| `GetOrCreateIndexReport`    | Implemented     | Get-then-create logic                                  |
| `HasIndexReport`            | Implemented     | Existence check via Clair                              |
| `GetVulnerabilities`        | Implemented     | Fetches from Clair, applies in-process enrichments     |
| `GetMetadata`               | Implemented     | Returns last vulnerability update time                 |
| `StoreIndexReport`          | Not implemented | Returns Unimplemented                                  |
| `GetRepositoryToCPEMapping` | Not implemented | Returns Unimplemented                                  |
| `GetSBOM`                   | Not implemented | Returns Unimplemented                                  |
| `ScanSBOM`                  | Not implemented | Returns Unimplemented                                  |

### Indexer

- **Registry interaction**: Fetches image manifests from container registries using `go-containerregistry`. Handles basic auth, insecure TLS skip, layer URI construction with auth headers for Clair.
- **Manifest metadata tracking**: Optional PostgreSQL store for tracking indexed manifests with TTL-based expiration.
- **Manifest GC**: Background garbage collector that deletes expired manifests from both the adapter database and Clair (via `DeleteIndexReport`). Configurable interval and throttle.

### Matcher

- **Vulnerability reports**: Fetches from Clair's HTTP API, returns enriched results.
- **Last vulnerability update**: Queries Clair's update operations endpoint or adapter's metadata store.

### In-Process Enrichment Pipeline

After receiving Clair's vulnerability report, the adapter applies these enrichments before returning results to Central:

| Enricher                  | Status                          | Notes                                                           |
|---------------------------|---------------------------------|-----------------------------------------------------------------|
| EPSS                      | Extracted from Clair's response | Clair provides natively                                         |
| NVD CVSS                  | Extracted from Clair's response | Clair provides natively                                         |
| CSAF (Red Hat advisories) | In-process                      | Matches RHSA/RHBA/RHEA names to advisory data                   |
| Fixed-by version          | In-process                      | Computes max fixed version per package (string comparison)      |
| Manual severity           | Parser implemented              | YAML parser for overrides; not yet wired to enrichment pipeline |

### Updater Pipeline

- **Online mode**: Fetches `vulnerabilities.zip` from `definitions.stackrox.io`, unpacks into per-bundle files.
- **Air-gapped mode**: Unpacks uploaded bundles from the same ZIP format.
- **HTTP server**: Serves vulnerability data at `/updater/{name}` and `/enricher/{name}` endpoints. Supports `ETag`/`If-None-Match` for conditional fetching. Clair is configured to poll these endpoints.
- **Periodic fetching**: Runs on configurable interval (default 5 minutes) with `If-Modified-Since` support.

### Infrastructure

- **mTLS**: Uses StackRox's `pkg/grpc` framework with `verifier.NonCA{}`. Loads certificates from `/run/secrets/stackrox.io/certs/` or configurable `certs_dir`. Compatible with Scanner V4's certificate chain.
- **Health endpoints**: `/healthz/live` (always 200), `/healthz/ready` (checks Clair connectivity). Served via `pkg/grpc` custom routes.
- **Configuration**: YAML-based with defaults. Fields: `clair_url`, `grpc_listen_addr`, `http_listen_addr`, `updater_listen_addr`, `vulnerabilities_url`, `certs_dir`, `indexer.enable`, `matcher.enable`, `log_level`.
- **Database**: PostgreSQL with connection retry (30 attempts). Tables for manifest metadata and vulnerability update tracking. SQL migration files included.
- **Deployment**: Dockerfile (UBI9 micro), `deploy-clair-adapter.sh` script for local k8s (auto-detects kind/minikube/Docker Desktop, deploys Clair DB + Clair + adapter).
- **Feature flag**: `ROX_CLAIR_ADAPTER` in `pkg/features/list.go`.

### Mappers

- **Index report**: Clair JSON `IndexReport` → `v4.IndexReport` proto (packages, distributions, repositories, environments).
- **Vulnerability report**: Clair JSON `VulnerabilityReport` → `v4.VulnerabilityReport` proto with enrichment integration (CVSS metrics, EPSS, CSAF, severity mapping, fixed-by).
- **Enrichment extraction**: Parses Clair's `enrichments` map for NVD, EPSS, CSAF, and fixed-by data.
- **Severity normalization**: Maps Clair severity strings to proto enum values.

### Test Coverage

164 tests across 16 packages. All unit tests use `httptest` mocks — no running Clair or PostgreSQL required.

---

## Remaining Gaps vs Scanner V4

### High Priority (blocks key workflows)

| Feature                           | Impact                                                                                                                                       | Effort Estimate                                                                                                                     |
|-----------------------------------|----------------------------------------------------------------------------------------------------------------------------------------------|-------------------------------------------------------------------------------------------------------------------------------------|
| **StoreIndexReport RPC**          | Central-initiated scans store external index reports for later matching. Without this, delegated scanning workflows don't work.              | Medium — port `ExternalIndexStore` from `scanner/datastore/postgres/external_index_store.go`, add version comparison logic.         |
| **GetRepositoryToCPEMapping RPC** | Sensor uses this to map RHEL repositories to CPEs for accurate vulnerability matching. Without it, RHEL vulnerability detection is degraded. | Medium — port `RepositoryToCPEFetcher` from `scanner/indexer/repositorytocpefetcher.go`, add HTTP caching with `If-Modified-Since`. |
| **GetSBOM RPC**                   | Returns an SPDX 2.3 JSON SBOM for an indexed image. Required for SBOM export features.                                                       | Large — requires porting `scanner/sbom/` (SBOM encoder, SPDX format support). ClairCore SBOM library is a retained dependency.      |
| **ScanSBOM RPC**                  | Accepts an SBOM and returns a vulnerability report. Required for SBOM-based scanning.                                                        | Large — requires porting SBOM decoder (`scanner/sbom/`), PURL registry, repo-to-CPE transform.                                      |

### Medium Priority (reduces quality or operational capability)

| Feature                        | Impact                                                                                                                                          | Effort Estimate                                                                                                  |
|--------------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------|------------------------------------------------------------------------------------------------------------------|
| **Distributed GC locks**       | Without distributed locks, multiple adapter replicas could GC simultaneously. Safe for single-replica but needed for HA.                        | Medium — port `ctxlock` usage from `scanner/indexer/manifest/manager.go`.                                        |
| **Remote indexer support**     | Scanner V4 can run indexer and matcher on separate pods. The adapter always runs both in one process.                                           | Medium — port `scanner/indexer/remote.go`, add `IndexerAddr` config.                                             |
| **Advanced fixed-by enricher** | Current implementation uses string comparison. Scanner V4 uses per-ecosystem version comparison (semver, Go semver, URL-encoded versions).      | Medium — port version comparison from `scanner/enricher/fixedby/fixedby.go` with matcher/version type detection. |
| **Readiness strategies**       | Scanner V4 supports `database` (ready when DB connected) and `vulnerability` (ready when vulns loaded). Adapter only checks Clair connectivity. | Small — add config field and logic.                                                                              |
| **Distribution notes**         | Scanner V4 adds `NOTE_OS_UNKNOWN` and `NOTE_OS_UNSUPPORTED` to vulnerability reports. Adapter doesn't.                                          | Small — port from `scanner/services/matcher.go`.                                                                 |
| **Transport mux**              | Scanner V4 routes mTLS traffic differently for Central vs Sensor endpoints. Adapter uses a single transport.                                    | Small — port from `scanner/internal/httputil/transport_mux.go`.                                                  |

### Low Priority (advanced/operational features)

| Feature                            | Impact                                                                                                                            | Effort Estimate                                                                            |
|------------------------------------|-----------------------------------------------------------------------------------------------------------------------------------|--------------------------------------------------------------------------------------------|
| **RC bundle support**              | Try release-candidate vulnerability bundles before GA. Only matters for pre-release testing.                                      | Small — add `EnableRCVulnBundle` config, URL resolution logic.                             |
| **Bundle allowlist**               | Restrict which vulnerability bundles are imported (e.g., only "alpine", "rhel-vex"). Reduces DB size.                             | Small — add config field, filter in updater.                                               |
| **Export functionality**           | Generate vulnerability bundles for distribution. Used in bundle production pipeline, not at runtime.                              | Medium — port `scanner/updater/export.go`.                                                 |
| **Proxy configuration**            | Dynamic proxy config with file watching. Adapter uses Go's default proxy env vars.                                                | Small — port `ProxyConfig` from scanner config.                                            |
| **Manifest migration**             | One-time migration for pre-metadata manifests during upgrades. Only needed for Scanner V4 → adapter upgrades.                     | Small — port from `scanner/indexer/manifest/manager.go`.                                   |
| **Versioned scanner tracking**     | Tracks which scanner versions indexed a manifest, detects obsolete scans. Used for re-indexing decisions.                         | Medium — port from `scanner/indexer/indexer.go`.                                           |
| **NVD/EPSS enricher data serving** | Currently Clair provides these natively. If Clair's native enrichers are insufficient, the adapter would need to serve this data. | Medium — only needed if Clair's built-in enrichers don't match Scanner V4's output format. |

---

## What's Identical

These aspects are fully at parity with Scanner V4:

- **gRPC proto contracts** — same `proto/internalapi/scanner/v4/` protos, Central/Sensor need zero changes
- **mTLS** — uses same `pkg/grpc` framework, same certificate paths, same CA chain
- **Vulnerability report format** — same proto fields, same severity enum, same CVSS metric structure
- **Index report format** — same proto fields for packages, distributions, repositories, environments
- **Feature flag pattern** — follows existing `pkg/features/` conventions
- **Deployment model** — separate service alongside Central, same port conventions (8443 gRPC, 9443 HTTP)

## Architecture Differences (by design, not gaps)

| Aspect                   | Scanner V4                                | Clair Adapter                                             |
|--------------------------|-------------------------------------------|-----------------------------------------------------------|
| **Vulnerability engine** | ClairCore embedded as Go library          | Upstream Clair as separate HTTP service                   |
| **Enrichment**           | ClairCore enricher framework (in-DB)      | Hybrid: Clair-native + in-process pipeline                |
| **Vulnerability data**   | Loaded directly into ClairCore's DB       | Adapter fetches and serves to Clair via HTTP              |
| **Database**             | Single PostgreSQL (shared with ClairCore) | Two databases: adapter's (metadata) + Clair's (vuln data) |
| **Clair dependency**     | ClairCore Go library (compile-time)       | Clair HTTP API (runtime)                                  |
| **Upgrade path**         | ClairCore version locked at compile time  | Clair version independently upgradeable                   |
