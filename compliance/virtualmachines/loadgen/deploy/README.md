# VM Load Generator Deployment

This directory contains Kubernetes manifests and configuration for deploying the vsock load generator.

## Files

- **`vsock-loadgen-daemonset.yaml`**: Complete DaemonSet deployment including:
  - ServiceAccount for node read permissions
  - ClusterRole/ClusterRoleBinding for CID range calculation
  - Role/RoleBinding for privileged SCC (OpenShift only)
  - DaemonSet configuration with vsock device access

- **`loadgen-config.yaml`**: Configuration file for load test parameters
  - Mounted as ConfigMap into the loadgen pods
  - Controls VM count, intervals, payload size, etc.

## Quick Deploy

### Using the helper script (recommended):

```bash
cd ../scripts
./run-loadgen.sh
```

### Manual deployment:

1. **Create the ConfigMap:**
   ```bash
   kubectl -n stackrox create configmap vsock-loadgen-config \
     --from-file=config.yaml=loadgen-config.yaml
   ```

2. **Deploy the DaemonSet:**
   ```bash
   kubectl apply -f vsock-loadgen-daemonset.yaml
   ```

3. **Verify deployment:**
   ```bash
   kubectl -n stackrox get pods -l app=vsock-loadgen -o wide
   kubectl -n stackrox logs -f -l app=vsock-loadgen --max-log-requests=5
   ```

## Configuration

Edit `loadgen-config.yaml` to adjust load test parameters. The most important settings:

```yaml
loadgen:
  # Total VMs across ALL nodes (distributed evenly)
  vmCount: 1000

  # How often each VM sends a report
  reportInterval: 60s

  # Payload size: small (~2.3MB), avg (~10MB), large (~50MB)
  payloadSize: small
```

After changing configuration, update the ConfigMap:

```bash
kubectl -n stackrox create configmap vsock-loadgen-config \
  --from-file=config.yaml=loadgen-config.yaml \
  --dry-run=client -o yaml | kubectl apply -f -

kubectl -n stackrox rollout restart daemonset/vsock-loadgen
```

## Architecture Notes

### DaemonSet Deployment

The load generator runs as a DaemonSet to distribute VMs across all worker nodes:

- One pod per worker node
- Automatic node affinity to exclude control plane nodes
- Each pod gets a unique CID range based on node index
- No CID conflicts between pods

### Resource Requirements

Each load generator pod requires:

- **Privileged access**: For vsock device (/dev/vsock)
- **Host network**: For vsock communication
- **Node read permissions**: For CID range calculation
- **CPU/Memory**: 500m CPU, 512Mi RAM (adjustable in manifest)

### Metrics

When enabled (`metricsPort: 9090`), each pod exposes Prometheus metrics:

```bash
# Forward metrics port from a specific pod
kubectl -n stackrox port-forward <pod-name> 9090:9090

# View metrics
curl http://localhost:9090/metrics
```

Available metrics:
- `vsock_loadgen_requests_total{result}`: Request counters by outcome
- `vsock_loadgen_bytes_total`: Total bytes sent
- `vsock_loadgen_request_latency_seconds`: Latency histogram

## Troubleshooting

### Pods not starting

```bash
# Check pod events
kubectl -n stackrox describe pod -l app=vsock-loadgen

# Check for vsock device
kubectl -n stackrox exec -it <pod-name> -- ls -la /dev/vsock
```

### Permission denied errors

On OpenShift, ensure the privileged SCC is bound:

```bash
kubectl get rolebinding -n stackrox vsock-loadgen-use-privileged-scc
```

### CID range conflicts

Check node assignments in logs:

```bash
kubectl -n stackrox logs -l app=vsock-loadgen | grep "assigned CID range"
```

Should show non-overlapping ranges:
```
Node worker-0 (index 0/3) assigned CID range [3-336] for 334 VMs
Node worker-1 (index 1/3) assigned CID range [10003-10335] for 333 VMs
Node worker-2 (index 2/3) assigned CID range [20003-20335] for 333 VMs
```

### No connection to relay

Verify relay is running and listening:

```bash
# Check relay logs
kubectl -n stackrox logs -l app.kubernetes.io/component=collector | grep vsock

# Check vsock port configuration
echo $ROX_VIRTUALMACHINES_VSOCK_PORT  # Should be 818
```

## Cleanup

```bash
# Delete DaemonSet
kubectl -n stackrox delete daemonset vsock-loadgen

# Delete ConfigMap
kubectl -n stackrox delete configmap vsock-loadgen-config

# Delete RBAC resources (optional)
kubectl delete clusterrole vsock-loadgen-node-reader
kubectl delete clusterrolebinding vsock-loadgen-node-reader
kubectl -n stackrox delete role vsock-loadgen-use-privileged-scc
kubectl -n stackrox delete rolebinding vsock-loadgen-use-privileged-scc
kubectl -n stackrox delete serviceaccount vsock-loadgen
```

## See Also

- [Main README](../README.md) - Load generator overview and usage
- [Scripts README](../scripts/README.md) - Build and deployment scripts
- `../../relay/` - The relay service that receives the load
