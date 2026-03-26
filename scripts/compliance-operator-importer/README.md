# CO → ACS Scheduled Scan Importer

Reads Compliance Operator `ScanSettingBinding` resources from one or more
Kubernetes clusters and creates equivalent scan configurations in Red Hat
Advanced Cluster Security (ACS) via the v2 API.

## Quick start

```bash
# Build
go build -o compliance-operator-importer ./cmd/importer

# Dry run (preview, no changes)
ROX_API_TOKEN=<token> ./compliance-operator-importer \
  --endpoint central.example.com \
  --dry-run

# Import for real
ROX_API_TOKEN=<token> ./compliance-operator-importer \
  --endpoint central.example.com
```

## Authentication

Auth mode is auto-inferred from environment variables:

| Variable | Mode | Typical use |
|----------|------|-------------|
| `ROX_API_TOKEN` | API token (Bearer) | Production |
| `ROX_ADMIN_PASSWORD` | Basic auth | Development/testing |

Setting both is an error. Setting neither is an error.

For basic auth the username defaults to `admin`; override with `--username`
or `ROX_ADMIN_USER`.

## Multi-cluster

By default all contexts in the merged kubeconfig are processed. Merge
multiple kubeconfig files via the standard `KUBECONFIG` variable:

```bash
KUBECONFIG=cluster-a.yaml:cluster-b.yaml ./compliance-operator-importer --endpoint central.example.com
```

Use `--context` (repeatable) to limit processing to specific contexts:

```bash
./compliance-operator-importer --endpoint central.example.com \
  --context prod-east \
  --context prod-west
```

When the same `ScanSettingBinding` name appears across multiple clusters,
the importer merges them into a single ACS scan configuration targeting all
matched clusters (profiles and schedules must match).

## Cluster ID auto-discovery

The ACS cluster ID for each context is auto-discovered using the first
successful method:

1. `admission-control` ConfigMap → `cluster-id` key (namespace: `stackrox`)
2. OpenShift `ClusterVersion` `spec.clusterID` → matched against ACS provider metadata
3. `helm-effective-cluster-name` Secret → matched against ACS cluster name

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--endpoint` | `ROX_ENDPOINT` | ACS Central URL (bare hostnames get `https://` prepended) |
| `--username` | `admin` | Basic auth username (`ROX_ADMIN_USER`) |
| `--context` | all | Kubeconfig context to process (repeatable) |
| `--co-namespace` | `openshift-compliance` | Namespace for CO resources |
| `--co-all-namespaces` | `false` | Read CO resources from all namespaces |
| `--dry-run` | `false` | Preview actions without changes |
| `--overwrite-existing` | `false` | Update existing ACS configs instead of skipping |
| `--report-json` | — | Write structured JSON report to file |
| `--max-retries` | `5` | Retry attempts for transient API errors (429, 502–504) |
| `--request-timeout` | `30s` | Per-request HTTP timeout |
| `--ca-cert-file` | — | PEM CA bundle for TLS |
| `--insecure-skip-verify` | `false` | Skip TLS verification |

## Behaviour

- **Create-only (default):** existing ACS scan configs with the same name
  are skipped with a warning.
- **Overwrite mode** (`--overwrite-existing`): existing configs are updated
  via PUT to match the cluster SSBs.
- **Idempotent:** re-running produces the same result; no duplicates.
- **Dry run:** all discovery and mapping runs normally; no POST/PUT issued.

## Exit codes

| Code | Meaning |
|------|---------|
| `0` | All bindings processed (or nothing to do) |
| `1` | Fatal error (config, auth, connectivity) |
| `2` | Partial success (some bindings failed; see report) |

## Mapping rules

Each `ScanSettingBinding` maps to one ACS scan configuration:

| ACS field | Source |
|-----------|--------|
| `scanName` | `ScanSettingBinding.metadata.name` |
| `profiles` | Sorted, deduplicated profile names from the binding |
| `scanSchedule` | Converted from the referenced `ScanSetting.schedule` (cron) |
| `clusters` | Auto-discovered ACS cluster ID(s) |
| `description` | `"Imported from CO ScanSettingBinding <ns>/<name> (ScanSetting: <ss>)"` |

Supported cron patterns: daily (`M H * * *`), weekly (`M H * * DOW`),
monthly (`M H DOM * *`). Step and range notation are not supported.

## JSON report

When `--report-json` is set, a structured report is written:

```json
{
  "meta": { "timestamp": "...", "dryRun": false, "mode": "create-only" },
  "counts": { "discovered": 3, "create": 2, "update": 0, "skip": 1, "failed": 0 },
  "items": [ { "source": {...}, "action": "create", ... } ],
  "problems": []
}
```

## Demo / testing

Seed demo fixtures (2 ACS scans + 3 SSBs, 1 conflicting):

```bash
ROX_ADMIN_PASSWORD=admin ROX_ENDPOINT=central.example.com ./hack/demo-seed.sh up
./hack/demo-seed.sh status
./hack/demo-seed.sh down
```

Interactive walkthrough:

```bash
ROX_ADMIN_PASSWORD=admin ROX_ENDPOINT=central.example.com ./hack/demo.sh
```

## Development

Specs live in `specs/` and are the source of truth. Tests reference spec IDs
(`IMP-*`). Run tests:

```bash
go test ./...
```
