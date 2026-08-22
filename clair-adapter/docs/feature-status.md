# Clair Adapter: Feature Status & Scanner V4 Gap Analysis

## What the Clair Adapter Is

The clair-adapter is a Go service that replaces Scanner V4 by delegating container image indexing and vulnerability matching to upstream Clair while preserving the Scanner V4 gRPC API. Central and Sensor connect to it using the same protos — it's a drop-in replacement.

Unlike Scanner V4 (which embeds ClairCore as a Go library), the adapter calls Clair over HTTP for indexing/matching and imports vulnerability data directly into Clair's PostgreSQL database.

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

### gRPC Security

- **Per-RPC authorization**: `indexerAuth` and `matcherAuth` rules matching Scanner V4's access control (Central+Sensor for indexer create/get, Central-only for store, Matcher-only for CPE mapping, Central-only for all matcher RPCs).
- **Identity extraction**: mTLS client certificate identity via `pkg/grpc/authn/service.NewExtractor()`.
- **Hash ID normalization**: Strips `/v4/containerimage/` prefix from StackRox hash IDs before calling Clair API.

### Indexer

- **Registry interaction**: Fetches image manifests from container registries using `go-containerregistry`. Handles basic auth, insecure TLS skip, layer URI construction with auth headers for Clair.
- **Manifest metadata tracking**: Optional PostgreSQL store for tracking indexed manifests with TTL-based expiration.
- **Manifest GC**: Background garbage collector that deletes expired manifests from both the adapter database and Clair (via `DeleteIndexReport`). Configurable interval and throttle.

### Matcher

- **Vulnerability reports**: Fetches from Clair's HTTP API, returns enriched results.
- **Last vulnerability update**: Queries adapter's metadata store (updated per-bundle after each import) or Clair's update operations endpoint as fallback.

### In-Process Enrichment Pipeline

After receiving Clair's vulnerability report, the adapter applies these enrichments before returning results to Central:

| Enricher                  | Status                          | Notes                                                           |
|---------------------------|---------------------------------|-----------------------------------------------------------------|
| EPSS                      | Extracted from Clair's response | Clair provides natively via imported data                       |
| NVD CVSS                  | Extracted from Clair's response | Clair provides natively via imported data                       |
| CSAF (Red Hat advisories) | In-process                      | Matches RHSA/RHBA/RHEA names to advisory data                   |
| Fixed-by version          | In-process                      | Computes max fixed version per package (string comparison)      |
| Manual severity           | Parser implemented              | YAML parser for overrides; not yet wired to enrichment pipeline |

### Vulnerability Data Pipeline

The adapter controls Clair's vulnerability data entirely — Clair's built-in updaters are disabled.

- **Direct DB import**: Fetches StackRox vulnerability bundles (zstd-compressed JSONL), decompresses, and imports directly into Clair's PostgreSQL via ClairCore's `datastore.MatcherStore` (`UpdateVulnerabilitiesIter`, `UpdateEnrichmentsIter`). Reuses `scanner/updater/jsonblob.Iterate()` for parsing.
- **Fingerprint dedup**: Skips unchanged bundles by comparing update operation fingerprints.
- **Central integration**: When deployed with StackRox, fetches bundles from Central's definitions endpoint (`/api/extensions/scannerdefinitions?version=...`) using mTLS. Falls back to `definitions.stackrox.io` CDN when Central is unavailable.
- **Schema readiness**: Waits for Clair's database schema (polls for `update_operation` table) before importing, handling the case where the adapter starts before Clair.
- **HTTP diagnostic server**: Keeps the updater HTTP server (`:9444`) as a diagnostic endpoint for inspecting loaded bundles, but Clair no longer polls it.

### Infrastructure

- **mTLS**: Uses StackRox's `pkg/grpc` framework with `verifier.NonCA{}`. Loads certificates from `/run/secrets/stackrox.io/certs/` or configurable `certs_dir`. Compatible with Scanner V4's certificate chain. The deploy script generates a `clair-adapter-tls` secret with SANs for both `scanner-v4-indexer` and `scanner-v4-matcher` DNS names.
- **mTLS HTTP client**: Fetcher uses `pkg/mtls.CACertPEM()` and `mtls.LeafCertificateFromFile()` to authenticate to Central's definitions endpoint.
- **Health endpoints**: `/healthz/live` (always 200), `/healthz/ready` (checks Clair connectivity). Served via `pkg/grpc` custom routes over HTTPS.
- **Configuration**: YAML-based with defaults. Fields: `clair_url`, `clair_db_connstring`, `grpc_listen_addr`, `http_listen_addr`, `updater_listen_addr`, `vulnerabilities_url`, `certs_dir`, `indexer.enable`, `matcher.enable`, `log_level`.
- **Database**: PostgreSQL with connection retry (30 attempts). Tables for manifest metadata and vulnerability update tracking. SQL migration files included.
- **Deployment**: Dockerfile (UBI9 micro), `deploy-clair-adapter.sh` script for local k8s. The script auto-detects kind/minikube/Docker Desktop, generates TLS certs from Central's CA, patches Scanner V4 services to route to the adapter, and auto-detects Central for the vulnerability data URL.
- **Feature flag**: `ROX_CLAIR_ADAPTER` in `pkg/features/list.go`.
- **NormalizedVersion parsing**: Custom `UnmarshalJSON` handles Clair's string format (e.g., `"dpkg:0:1.19.7"`) in addition to the struct format.

### Mappers

- **Index report**: Clair JSON `IndexReport` → `v4.IndexReport` proto (packages, distributions, repositories, environments).
- **Vulnerability report**: Clair JSON `VulnerabilityReport` → `v4.VulnerabilityReport` proto with enrichment integration (CVSS metrics, EPSS, CSAF, severity mapping, fixed-by).
- **Enrichment extraction**: Parses Clair's `enrichments` map for NVD, EPSS, CSAF, and fixed-by data.
- **Severity normalization**: Maps Clair severity strings to proto enum values.

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
- **Per-RPC authorization** — same access control rules as Scanner V4 (Central, Sensor, Matcher identity checks)
- **Vulnerability report format** — same proto fields, same severity enum, same CVSS metric structure
- **Index report format** — same proto fields for packages, distributions, repositories, environments
- **Vulnerability data pipeline** — same bundle format, same JSONL parsing, same fingerprint-based dedup
- **Feature flag pattern** — follows existing `pkg/features/` conventions
- **Deployment model** — separate service alongside Central, same port conventions (8443 gRPC, 9443 HTTP)

## Architecture Differences (by design, not gaps)

| Aspect                   | Scanner V4                                       | Clair Adapter                                                     |
|--------------------------|--------------------------------------------------|-------------------------------------------------------------------|
| **Vulnerability engine** | ClairCore embedded as Go library                 | Upstream Clair as separate HTTP service                           |
| **Enrichment**           | ClairCore enricher framework (in-DB, in-process) | Hybrid: data imported into Clair DB + in-process pipeline         |
| **Vulnerability data**   | Loaded directly into ClairCore's DB via library  | Adapter imports into Clair's DB via ClairCore datastore interface |
| **Database**             | Single PostgreSQL (shared with ClairCore)        | Two databases: adapter's (metadata) + Clair's (vuln data)         |
| **Clair dependency**     | ClairCore Go library (compile-time)              | Clair HTTP API (runtime) + ClairCore datastore (compile-time)     |
| **Upgrade path**         | ClairCore version locked at compile time         | Clair version independently upgradeable                           |
