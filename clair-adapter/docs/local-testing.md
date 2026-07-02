# Clair Adapter: Local E2E Testing Guide

## Overview

This guide covers testing the clair-adapter end-to-end on a local Kubernetes cluster. The adapter replaces Scanner V4 by delegating to upstream Clair for indexing and vulnerability matching, importing vulnerability data directly into Clair's database, and applying StackRox-specific enrichments in-process.

## Prerequisites

- Local Kubernetes cluster (Docker Desktop, minikube, or kind)
- `kubectl` configured for your cluster
- `docker` CLI for building images
- `grpcurl` for gRPC testing (`brew install grpcurl`)
- Go 1.25+ for building the adapter binary

## Architecture

```
                    gRPC (mTLS)
Central/Sensor ────────────────▶ Clair Adapter ──HTTP──▶ Upstream Clair
                                      │                       │
                                      │                       ▼
                              mTLS    │               Clair PostgreSQL
                       ┌──────────────┘                    ▲
                       ▼                                   │
                    Central                         Direct DB import
             /api/extensions/                     (ClairCore datastore)
            scannerdefinitions                         │
                       │                               │
                       ▼                               │
               vulnerabilities.zip ──── unpack ────────┘
```

The adapter:
- Listens on `:8443` (gRPC, mTLS) for Central/Sensor connections
- Listens on `:9443` (HTTPS) for health checks
- Listens on `:9444` (HTTP) for diagnostic updater endpoint
- Fetches vulnerability bundles from Central (or `definitions.stackrox.io` fallback)
- Imports vulnerability data directly into Clair's PostgreSQL
- Fetches container image manifests from registries and submits to Clair for indexing
- Applies CSAF, fixed-by, and manual severity enrichments in-process

## Quick Start

### 1. Build the adapter image

```bash
cd /Users/blugo/dev/stackrox/stackrox/scanner-adapter

# Build the container image
docker build -t clair-adapter:dev -f clair-adapter/Dockerfile .

# Load into your cluster
# For kind:
kind load docker-image clair-adapter:dev
# For minikube:
minikube image load clair-adapter:dev
# For Docker Desktop: image is already available
```

### 2. Deploy StackRox + Clair Adapter

The recommended flow deploys StackRox first, then deploys the adapter alongside it:

```bash
# 1. Deploy StackRox normally
./deploy/deploy-local.sh

# 2. Deploy Clair + Adapter (auto-detects Central, generates TLS certs, patches services)
BUILD_IMAGE=true ./clair-adapter/deploy/deploy-clair-adapter.sh

# The script will:
# - Build the adapter image
# - Scale down Scanner V4
# - Patch scanner-v4-indexer/matcher services to route to the adapter
# - Generate a TLS cert signed by Central's CA with Scanner V4 DNS SANs
# - Deploy Clair DB + Clair + Adapter
# - Configure the adapter to fetch vuln data from Central
# - Wait for all pods to be ready
```

### 3. Verify

```bash
# Check pods are running
kubectl get pods -n stackrox -l app.kubernetes.io/part-of=clair-adapter

# Check adapter logs for successful bundle import
kubectl logs -n stackrox deploy/clair-adapter | grep -i "imported\|import"

# Test via Central UI or roxctl
kubectl port-forward -n stackrox svc/central 8000:443 &
roxctl -e localhost:8000 -p <admin-password> image scan --image docker.io/library/alpine:3.18
```

### 4. Test with grpcurl (without StackRox)

If testing without StackRox deployed, the adapter runs without mTLS:

```bash
# Deploy without StackRox (uses definitions.stackrox.io CDN for vuln data)
SCALE_DOWN_SCANNER_V4=false ./clair-adapter/deploy/deploy-clair-adapter.sh

kubectl port-forward -n stackrox svc/clair-adapter 8443:8443 &

# List available services
grpcurl -plaintext localhost:8443 list

# Index a real image
grpcurl -plaintext localhost:8443 scanner.v4.Indexer/CreateIndexReport \
  -d '{
    "hash_id": "/v4/containerimage/sha256:test123",
    "container_image": {
      "url": "https://docker.io/library/alpine:3.18"
    }
  }'

# Get vulnerabilities for the indexed image
grpcurl -plaintext localhost:8443 scanner.v4.Matcher/GetVulnerabilities \
  -d '{"hash_id": "/v4/containerimage/sha256:test123"}'

# Get vulnerability metadata (last update time)
grpcurl -plaintext localhost:8443 scanner.v4.Matcher/GetMetadata
```

When deployed with StackRox, mTLS is required. Use grpcurl with certs:

```bash
# Extract certs from the adapter's TLS secret
kubectl get secret clair-adapter-tls -n stackrox -o jsonpath='{.data.ca\.pem}' | base64 -d > /tmp/ca.pem
kubectl get secret clair-adapter-tls -n stackrox -o jsonpath='{.data.cert\.pem}' | base64 -d > /tmp/cert.pem
kubectl get secret clair-adapter-tls -n stackrox -o jsonpath='{.data.key\.pem}' | base64 -d > /tmp/key.pem

kubectl port-forward -n stackrox svc/clair-adapter 8443:8443 &

grpcurl -cacert /tmp/ca.pem -cert /tmp/cert.pem -key /tmp/key.pem \
  localhost:8443 scanner.v4.Indexer/HasIndexReport \
  -d '{"hash_id": "/v4/containerimage/sha256:test"}'
```

## Configuration

The adapter is configured via YAML. Key fields:

| Field                  | Default                          | Description                                         |
|------------------------|----------------------------------|-----------------------------------------------------|
| `clair_url`            | `http://localhost:8080`          | Upstream Clair HTTP endpoint                        |
| `clair_db_connstring`  | (empty)                          | Clair's PostgreSQL connection string for direct import |
| `grpc_listen_addr`     | `:8443`                          | gRPC endpoint (mTLS)                                |
| `http_listen_addr`     | `:9443`                          | Health/HTTP endpoint (HTTPS)                        |
| `updater_listen_addr`  | `:9444`                          | Diagnostic updater HTTP endpoint                    |
| `vulnerabilities_url`  | Central's endpoint or CDN        | Where to fetch vulnerability bundles                |
| `certs_dir`            | `/run/secrets/stackrox.io/certs` | mTLS certificate directory                          |
| `indexer.enable`       | `true`                           | Enable indexer service                              |
| `matcher.enable`       | `true`                           | Enable matcher service                              |

## How the Deploy Script Works

`clair-adapter/deploy/deploy-clair-adapter.sh` does the following:

1. **Optionally builds** the adapter image (`BUILD_IMAGE=true`)
2. **Loads image** into cluster (kind/minikube/Docker Desktop auto-detected)
3. **Scales down Scanner V4** indexer and matcher deployments
4. **Patches Scanner V4 services** — changes `scanner-v4-indexer` and `scanner-v4-matcher` service selectors to route to the adapter pod, so Central connects to the adapter at the same DNS names
5. **Generates TLS certificate** — extracts CA from `central-tls` secret, creates a cert with SANs for both `scanner-v4-indexer` and `scanner-v4-matcher` DNS names, stores as `clair-adapter-tls` secret
6. **Deploys Clair DB** — PostgreSQL 15 with trust auth
7. **Deploys upstream Clair** — combo mode, native updaters disabled
8. **Deploys clair-adapter** — configured with Clair URL, Clair DB connection, Central vuln URL, TLS certs
9. **Detects image changes** — restarts adapter if the local image differs from what's running
10. **Waits for readiness** — all three deployments must be ready

## Troubleshooting

### Check adapter logs
```bash
kubectl logs -n stackrox deploy/clair-adapter -f
```

### Check Clair logs
```bash
kubectl logs -n stackrox deploy/clair -f
```

### Adapter not ready
The adapter's readiness check pings Clair's index state endpoint. If Clair isn't up yet, the adapter will report not ready. Check Clair DB connectivity:
```bash
kubectl logs -n stackrox deploy/clair-db
kubectl logs -n stackrox deploy/clair
```

### No vulnerabilities returned
The adapter imports vulnerability data directly into Clair's PostgreSQL. Check:
```bash
# Look for successful import messages
kubectl logs -n stackrox deploy/clair-adapter | grep -i "imported"

# Verify Clair's DB has vulnerability data
kubectl exec -n stackrox deploy/clair-db -- psql -U clair -c "SELECT count(*) FROM vuln"
```

If the adapter fetches from Central, verify Central is serving definitions:
```bash
kubectl logs -n stackrox deploy/central | grep -i "scannerdefinitions"
```

### Image indexing fails
Registry authentication issues are the most common cause:
```bash
kubectl logs -n stackrox deploy/clair-adapter | grep -i "registry\|layer\|index"
kubectl logs -n stackrox deploy/clair | grep -i "index\|layer\|fetch"
```

### mTLS connection failures
```bash
# Verify the TLS secret exists and has the right SANs
kubectl get secret clair-adapter-tls -n stackrox
openssl x509 -in <(kubectl get secret clair-adapter-tls -n stackrox -o jsonpath='{.data.cert\.pem}' | base64 -d) -text -noout | grep DNS
```

## Current Limitations

- **No SBOM**: `GetSBOM` and `ScanSBOM` RPCs return Unimplemented.
- **No Helm chart**: The adapter is deployed via raw manifests, not integrated into StackRox's Helm charts.
- **No StoreIndexReport**: Delegated scanning workflows that store external index reports are not yet supported.
- **Simple fixed-by**: Uses string comparison instead of per-ecosystem version comparison.

## Cleanup

```bash
# Remove Clair + Adapter
kubectl delete deploy clair-adapter clair clair-db -n stackrox
kubectl delete svc clair-adapter clair clair-db -n stackrox
kubectl delete configmap clair-config clair-adapter-config -n stackrox
kubectl delete secret clair-adapter-tls -n stackrox 2>/dev/null

# Restore Scanner V4 services (if you patched them)
kubectl scale deploy scanner-v4-indexer scanner-v4-matcher -n stackrox --replicas=1
# You may also need to restore the original service selectors
```
