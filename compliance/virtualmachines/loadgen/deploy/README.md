# Deploy

Kubernetes manifests for the vsock load generator.

## Files

- `vsock-loadgen-daemonset.yaml` - DaemonSet, ServiceAccount, RBAC
- `loadgen-config.yaml` - Load test configuration

## Deploy

Using the helper script (recommended):

```bash
cd ../scripts && ./run-loadgen.sh
```

Manual deployment:

```bash
# Create ConfigMap
kubectl -n stackrox create configmap vsock-loadgen-config \
  --from-file=config.yaml=loadgen-config.yaml

# Deploy DaemonSet (requires envsubst for $USER substitution)
envsubst < vsock-loadgen-daemonset.yaml | kubectl apply -f -
```

## Update Configuration

```bash
kubectl -n stackrox create configmap vsock-loadgen-config \
  --from-file=config.yaml=loadgen-config.yaml \
  --dry-run=client -o yaml | kubectl apply -f -

kubectl -n stackrox rollout restart daemonset/vsock-loadgen
```

## Cleanup

```bash
kubectl -n stackrox delete daemonset vsock-loadgen
kubectl -n stackrox delete configmap vsock-loadgen-config
kubectl delete clusterrole vsock-loadgen-node-reader
kubectl delete clusterrolebinding vsock-loadgen-node-reader
kubectl -n stackrox delete serviceaccount vsock-loadgen
```
