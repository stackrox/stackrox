# VM Load Generator

Simulates VMs sending index reports via vsock to test the VM compliance pipeline at scale.

## Quick Start

```bash
# Build and push image
cd scripts && ./build-loadgen.sh

# Edit config
vi deploy/loadgen-config.yaml

# Deploy
cd scripts && ./run-loadgen.sh

# Monitor
kubectl -n stackrox logs -f -l app=vsock-loadgen

# Cleanup
kubectl -n stackrox delete daemonset vsock-loadgen
kubectl -n stackrox delete configmap vsock-loadgen-config
kubectl delete clusterrole vsock-loadgen-node-reader
kubectl delete clusterrolebinding vsock-loadgen-node-reader
kubectl -n stackrox delete serviceaccount vsock-loadgen

```

## Configuration

Edit `deploy/loadgen-config.yaml`:

```yaml
loadgen:
  vmCount: 1000           # Total VMs across all nodes
  reportInterval: 60s     # How often each VM reports
  numPackages: 700        # Packages per report (controls payload size)
  statsInterval: 30s
  port: 818
  metricsPort: 9090       # 0 to disable
  requestTimeout: 10s
```

## How It Works

- Deployed as DaemonSet across worker nodes
- Each pod simulates multiple VMs (goroutines) with unique CIDs
- CID ranges are automatically partitioned (max 10,000 per node)
- Pre-generates payloads at startup for zero per-request overhead
- Sends protobuf-encoded index reports over vsock to the relay

### CID Assignment

Nodes are sorted alphabetically; each gets a non-overlapping CID range:
- Node 0: CIDs 3-10002
- Node 1: CIDs 10003-20002
- Node 2: CIDs 20003-30002

## Metrics

When `metricsPort` is enabled:

```bash
kubectl -n stackrox port-forward daemonset/vsock-loadgen 9090:9090
curl http://localhost:9090/metrics
```

Available metrics:
- `vsock_loadgen_requests_total{result}` - request counts
- `vsock_loadgen_bytes_total` - bytes sent
- `vsock_loadgen_request_latency_seconds` - latency histogram

## Troubleshooting

```bash
# Check pod logs
kubectl -n stackrox logs -l app=vsock-loadgen

# Verify CID assignments
kubectl -n stackrox logs -l app=vsock-loadgen | grep "assigned CID range"

# Check pod events
kubectl -n stackrox describe pod -l app=vsock-loadgen
```
