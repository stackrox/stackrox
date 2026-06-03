# Clair Adapter: Local E2E Testing Guide

## Overview

This guide covers testing the clair-adapter end-to-end on a local Kubernetes cluster. The adapter replaces Scanner V4 by delegating to upstream Clair for indexing and vulnerability matching, while applying StackRox-specific enrichments in-process.

## Prerequisites

- Local Kubernetes cluster (Docker Desktop, minikube, or kind)
- `kubectl` configured for your cluster
- `docker` CLI for building images
- `grpcurl` for gRPC testing (`brew install grpcurl`)
- Go 1.24+ for building the adapter binary

## Architecture

```
Central ──gRPC──▶ Clair Adapter ──HTTP──▶ Upstream Clair
                       │                       │
                       │                       ▼
                       │               Clair PostgreSQL
                       │
                       ▼
               Adapter PostgreSQL
               (manifest metadata)
```

The adapter:
- Listens on `:8443` (gRPC) for Central/Sensor connections
- Listens on `:9443` (HTTP) for health checks
- Listens on `:9444` (HTTP) for serving vulnerability data to Clair
- Fetches vulnerability data from `definitions.stackrox.io` and serves it to Clair
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

### 2. Deploy Clair + Adapter

```bash
# Deploy everything (Clair DB, Clair, Adapter DB, Adapter)
./clair-adapter/deploy/deploy-clair-adapter.sh

# Wait for pods
kubectl get pods -n stackrox -w
```

### 3. Verify

```bash
# Check health
kubectl port-forward -n stackrox svc/clair-adapter 9443:9443 &
curl http://localhost:9443/healthz/live    # Should return 200
curl http://localhost:9443/healthz/ready   # 200 when Clair is up

# Check updater is serving data
kubectl port-forward -n stackrox svc/clair-adapter 9444:9444 &
curl -s -o /dev/null -w "%{http_code}" http://localhost:9444/updater/alpine
# Should return 200 (or 404 if bundles haven't been fetched yet)
```

### 4. Test with grpcurl

```bash
kubectl port-forward -n stackrox svc/clair-adapter 8443:8443 &

# List available services
grpcurl -plaintext localhost:8443 list

# Check if an index report exists (expect not found)
grpcurl -plaintext localhost:8443 scanner.v4.Indexer/HasIndexReport \
  -d '{"hash_id": "sha256:nonexistent"}'

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

## Testing with StackRox Central

### Option A: Deploy StackRox first, then swap scanner

```bash
# 1. Deploy StackRox normally
./deploy/deploy-local.sh

# 2. Deploy Clair + Adapter alongside
./clair-adapter/deploy/deploy-clair-adapter.sh

# 3. Scale down Scanner V4
kubectl scale deploy scanner-v4-indexer scanner-v4-matcher -n stackrox --replicas=0

# 4. Point Central at the adapter (no mTLS — only works for basic testing)
# Central expects the scanner endpoint via internal config.
# For a quick test, port-forward and use roxctl:
kubectl port-forward -n stackrox svc/central 8000:443 &
roxctl -e localhost:8000 -p <admin-password> image scan --image docker.io/library/alpine:3.18
```

### Option B: Deploy with Scanner V4 disabled

```bash
# Deploy StackRox without Scanner V4
export ROX_SCANNER_V4=false
./deploy/deploy-local.sh

# Then deploy Clair + Adapter
./clair-adapter/deploy/deploy-clair-adapter.sh
```

Note: Central won't automatically connect to the adapter in this mode because it expects Scanner V4's mTLS certificates and service discovery. Full Central integration requires Helm chart changes (future work).

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

### Vulnerability data not available
The adapter fetches vulnerability bundles from `definitions.stackrox.io`. Check:
```bash
# Verify fetcher is running
kubectl logs -n stackrox deploy/clair-adapter | grep -i "fetch\|vuln\|bundle"

# Check if the updater server has data
kubectl port-forward -n stackrox svc/clair-adapter 9444:9444 &
curl -v http://localhost:9444/updater/alpine
```

### Image indexing fails
Registry authentication issues are the most common cause. Check:
```bash
kubectl logs -n stackrox deploy/clair-adapter | grep -i "registry\|layer\|index"
kubectl logs -n stackrox deploy/clair | grep -i "index\|layer\|fetch"
```

## Current Limitations

- **No mTLS**: The adapter doesn't implement mTLS yet, so Central can't connect to it in a standard StackRox deployment without additional configuration.
- **No SBOM**: `GetSBOM` and `ScanSBOM` RPCs return Unimplemented.
- **No Helm chart**: The adapter is deployed via raw manifests, not integrated into StackRox's Helm charts.
- **Clair updaters disabled**: Vulnerability data depends entirely on the adapter's fetcher downloading from `definitions.stackrox.io`.

## Cleanup

```bash
# Remove Clair + Adapter
kubectl delete deploy clair-adapter clair clair-db clair-adapter-db -n stackrox
kubectl delete svc clair-adapter clair clair-db clair-adapter-db -n stackrox
kubectl delete configmap clair-config clair-adapter-config -n stackrox
kubectl delete pvc clair-db-data clair-adapter-db-data -n stackrox 2>/dev/null
```
