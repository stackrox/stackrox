# VM Load Generator

A load testing tool for StackRox VM compliance infrastructure that simulates VMs sending index reports via vsock.

## Overview

The VM load generator simulates hundreds to thousands of virtual machines sending index reports to the relay service via vsock connections. It's designed to test the scalability and performance of the VM compliance pipeline under realistic load conditions.

### What It Simulates

- **VM Behavior**: Each goroutine simulates a single VM with a unique CID (Context Identifier)
- **Periodic Reporting**: VMs send index reports at configurable intervals with realistic timing jitter
- **Realistic Payloads**: Pre-generates unique index reports for each VM with configurable sizes (small/avg/large)
- **Distributed Load**: Deployed as a DaemonSet across worker nodes with automatic CID range partitioning

## Architecture

### Components

- **Main Binary** (`main.go`): The load generator binary that runs in each DaemonSet pod
- **Deploy Manifests** (`deploy/`): Kubernetes manifests for deploying the load generator
  - `vsock-loadgen-daemonset.yaml`: DaemonSet, ServiceAccount, RBAC configuration
  - `loadgen-config.yaml`: ConfigMap with load test parameters
- **Scripts** (`scripts/`): Helper scripts for building and deploying
  - `build-loadgen.sh`: Builds the binary and container image, pushes to registry
  - `run-loadgen.sh`: Deploys the load generator to the cluster

### CID Assignment (No Overlap)

When deployed as a DaemonSet, each pod automatically calculates a unique CID range based on its node's position in the cluster:

- Nodes are sorted alphabetically by name for deterministic ordering
- Each node gets a partition with 10,000 CID spacing to prevent overlap:
  - Node 0: starts at CID 3
  - Node 1: starts at CID 10003
  - Node 2: starts at CID 20003

Example: With 3 worker nodes and `vmCount=1000`:
- Node "worker-0" (index 0): 334 VMs, CIDs 3-336
- Node "worker-1" (index 1): 333 VMs, CIDs 10003-10335
- Node "worker-2" (index 2): 333 VMs, CIDs 20003-20335

## Usage

### Prerequisites

- Running StackRox deployment with VM compliance enabled
- Docker/Podman for building images
- Access to a container registry (e.g., quay.io)
- kubectl access to the cluster

### Quick Start

1. **Build and push the load generator image:**
   ```bash
   cd compliance/virtualmachines/loadgen/scripts
   ./build-loadgen.sh
   ```

2. **Configure the load test** (edit `deploy/loadgen-config.yaml`):
   ```yaml
   loadgen:
     vmCount: 1000          # Total VMs across all nodes
     reportInterval: 60s    # How often each VM reports
     payloadSize: small     # small (~2.3MB), avg (~10MB), large (~50MB)
     statsInterval: 30s     # How often to print stats
   ```

3. **Deploy the load generator:**
   ```bash
   ./run-loadgen.sh
   ```

4. **Monitor the load:**
   ```bash
   # View logs from all pods
   kubectl -n stackrox logs -f -l app=vsock-loadgen --max-log-requests=10

   # Check Prometheus metrics (if enabled)
   kubectl -n stackrox port-forward daemonset/vsock-loadgen 9090:9090
   # Visit: http://localhost:9090/metrics
   ```

5. **Stop and cleanup:**
   ```bash
   kubectl -n stackrox delete daemonset vsock-loadgen
   kubectl -n stackrox delete configmap vsock-loadgen-config
   ```

### Advanced Usage

#### Custom Build Options

```bash
# Build locally without pushing
./build-loadgen.sh --no-push

# Push without restarting DaemonSet
./build-loadgen.sh --no-restart

# Use custom image repository
export VSOCK_LOADGEN_IMAGE="quay.io/myorg/vsock-loadgen"
export VSOCK_LOADGEN_TAG="v1.0"
./build-loadgen.sh
```

#### Custom Configuration

```bash
# Use a custom config file
./run-loadgen.sh /path/to/custom-config.yaml
```

## Configuration Reference

### `loadgen-config.yaml`

- **`vmCount`**: Total number of VMs to simulate across ALL nodes (max: 100,000)
- **`reportInterval`**: Interval at which each VM sends reports (e.g., 30s, 1m, 5m)
- **`payloadSize`**: Size of index reports
  - `small`: ~2.3MB (514 packages)
  - `avg`: ~10MB (700 packages)
  - `large`: ~50MB (1500 packages)
- **`statsInterval`**: How often to print statistics to logs (e.g., 30s, 1m)
- **`port`**: Vsock port to connect to (default: 818, relay's listening port)
- **`metricsPort`**: Prometheus metrics port (default: 9090, 0 to disable)
- **`requestTimeout`**: Per-request vsock deadline (default: 10s)

## Performance Optimization

The load generator uses several optimizations for high throughput:

1. **Pre-generation**: All index reports are generated and marshaled at startup
2. **No per-request overhead**: Eliminates protobuf cloning and marshaling during load test
3. **Realistic timing**: Random initial delays and jittered intervals prevent thundering herd
4. **Error rate limiting**: Prevents log spam from overwhelming the cluster

## Metrics

When `metricsPort` is enabled (default: 9090), the following Prometheus metrics are exposed:

- `vsock_loadgen_requests_total{result}`: Total requests by result (success/dial/write/etc.)
- `vsock_loadgen_bytes_total`: Total bytes sent to the relay
- `vsock_loadgen_request_latency_seconds`: Request latency histogram

## Troubleshooting

### Pods not starting

Check if vsock device is available:
```bash
kubectl -n stackrox logs -l app=vsock-loadgen
```

### CID range conflicts

Check node assignments:
```bash
kubectl -n stackrox logs -l app=vsock-loadgen | grep "assigned CID range"
```

### Low throughput

- Increase resources in the DaemonSet manifest
- Check relay service logs for bottlenecks
- Verify network connectivity between pods and relay

### Metrics not available

Ensure port-forward is set up correctly:
```bash
kubectl -n stackrox get pods -l app=vsock-loadgen
kubectl -n stackrox port-forward <pod-name> 9090:9090
```

## Development

### Building from source

```bash
cd /path/to/stackrox
make compliance/virtualmachines/loadgen
```

### Running locally

```bash
# Requires vsock device and relay running
./bin/linux_amd64/loadgen \
  --vm-count 10 \
  --report-interval 30s \
  --payload-size small \
  --port 818
```

## Related Components

- **Relay** (`../relay/`): The vsock relay service that receives index reports
- **roxagent** (`../roxagent/`): The agent running inside VMs that creates real index reports
