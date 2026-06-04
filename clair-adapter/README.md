# Clair Adapter

The clair-adapter is a Go service that replaces Scanner V4 by delegating container image indexing and vulnerability matching to [upstream Clair](https://github.com/quay/clair) while preserving the Scanner V4 gRPC API. Central and Sensor connect to it using the same protos, making it a drop-in replacement.

Unlike Scanner V4 (which embeds ClairCore as a Go library), the adapter calls Clair over HTTP for indexing/matching and imports StackRox vulnerability data directly into Clair's PostgreSQL database.

## Architecture

```
Central/Sensor ──gRPC (mTLS)──> Clair Adapter ──HTTP──> Upstream Clair
                                     │                       │
                                     │                       v
                                     │               Clair PostgreSQL
                                     │                  (shared)
                                     │
                                     ├── Fetches vuln bundles from Central
                                     ├── Imports vulns directly into Clair's DB
                                     └── Queries NVD/EPSS enrichments from DB
```

### Ports

| Port | Protocol | Purpose |
|------|----------|---------|
| 8443 | gRPC     | Scanner V4 API (Central/Sensor connections) |
| 9443 | HTTPS    | Health checks (`/healthz/live`, `/healthz/ready`) |
| 9444 | HTTP     | Vulnerability bundle server (diagnostic) |

## Quick Start

Prerequisites: a running Kubernetes cluster with StackRox deployed via `deploy/deploy-local.sh`.

```bash
# Build and deploy everything (Clair DB, Clair, adapter)
BUILD_IMAGE=true ./clair-adapter/deploy/deploy-clair-adapter.sh
```

The deploy script handles:
- Building the adapter container image
- Generating a combined TLS certificate for both scanner-v4-indexer and scanner-v4-matcher identities
- Scaling down Scanner V4 and patching its services to route to the adapter
- Deploying Clair PostgreSQL, upstream Clair, and the adapter
- Detecting config changes and restarting Clair when needed
- Comparing image SHAs to avoid unnecessary adapter restarts

## Configuration

The adapter is configured via a YAML file (passed with `-config`):

```yaml
clair_url: "http://clair.stackrox.svc:8080"
clair_db_connstring: "host=clair-db.stackrox.svc port=5432 user=clair dbname=clair sslmode=disable"
grpc_listen_addr: "0.0.0.0:8443"
http_listen_addr: "0.0.0.0:9443"
updater_listen_addr: "0.0.0.0:9444"
vulnerabilities_url: "https://central.stackrox.svc/api/extensions/scannerdefinitions?version=dev"
certs_dir: "/run/secrets/stackrox.io/certs"
indexer:
  enable: true
matcher:
  enable: true
```

| Field | Description |
|-------|-------------|
| `clair_url` | Upstream Clair HTTP endpoint |
| `clair_db_connstring` | Clair's PostgreSQL connection string (for direct vuln/enrichment import) |
| `vulnerabilities_url` | Where to fetch vulnerability bundles (Central's endpoint or definitions.stackrox.io) |
| `certs_dir` | Path to StackRox mTLS certificates (ca.pem, cert.pem, key.pem) |

## Vulnerability Data Pipeline

The adapter controls the full vulnerability data pipeline:

1. **Fetch**: Downloads StackRox vulnerability bundles from Central (`/api/extensions/scannerdefinitions`)
2. **Import**: Streams bundles from disk through zstd decompression and JSONL parsing, then inserts vulnerabilities and enrichments directly into Clair's PostgreSQL using ClairCore's `MatcherStore`
3. **Match**: When Central requests a vulnerability scan, the adapter queries Clair's matcher API
4. **Enrich**: Queries NVD (CVSS scores) and EPSS (exploit probabilities) directly from the enrichment tables, bypassing Clair's enrichment pipeline which lacks the StackRox enricher plugins

See [docs/vulnerability-data-pipeline.md](docs/vulnerability-data-pipeline.md) for details.

## Package Structure

| Package | Purpose |
|---------|---------|
| `cmd/clair-adapter` | Entrypoint, wires all components |
| `clairclient` | HTTP client for upstream Clair's indexer/matcher APIs |
| `config` | YAML configuration loading |
| `enricher` | Enrichment pipeline (CSAF, NVD, EPSS, fixedby) |
| `healthz` | HTTP health check handlers |
| `indexer` | Container image indexing via Clair |
| `mappers` | Converts between Clair types and Scanner V4 protos |
| `matcher` | Vulnerability matching with enrichment injection |
| `services` | gRPC service implementations (IndexerService, MatcherService) |
| `updater` | Vulnerability bundle fetching, unpacking, and HTTP serving |
| `vulnimporter` | Direct PostgreSQL import of vulnerabilities and enrichment queries |

## mTLS

The adapter uses a single TLS certificate with dual identity to serve both Scanner V4 Indexer and Matcher roles:

- **CN**: `SCANNER_V4_INDEXER_SERVICE: Scanner V4 Indexer` (passes Central's CN check for indexer connections)
- **OU**: `SCANNER_V4_MATCHER_SERVICE` (passes Central's OU fallback check for matcher connections)
- **SANs**: `scanner-v4-indexer.stackrox.svc`, `scanner-v4-matcher.stackrox.svc` (plus `.svc.cluster.local` variants)

The deploy script generates this certificate automatically from the StackRox CA in the `central-tls` secret.

## Further Documentation

- [Local E2E Testing Guide](docs/local-testing.md)
- [Feature Status & Scanner V4 Gap Analysis](docs/feature-status.md)
- [Vulnerability Data Pipeline](docs/vulnerability-data-pipeline.md)
