# Remote Cluster Support for Sensor

This package implements support for connecting Sensor to a remote Kubernetes cluster using configuration stored in a Kubernetes secret.

## Overview

Similar to the fake workload generator in `sensor/kubernetes/fake`, the remote cluster feature allows Sensor to read Kubernetes resources from a different cluster than the one it's running in. This is useful for:

- Testing Sensor behavior against remote clusters
- Monitoring clusters that are not directly accessible
- Development and debugging scenarios

## Configuration

The remote cluster feature is controlled by environment variables:

- `ROX_REMOTE_CLUSTER_SECRET`: Name of the secret containing the kubeconfig for the remote cluster
- `ROX_REMOTE_CLUSTER_SECRET_NAMESPACE`: Namespace where the secret is located (defaults to Sensor's namespace)

## Usage

### 1. Create a kubeconfig for the remote cluster

Generate a kubeconfig file that has the necessary permissions to access the remote cluster's API server.

### 2. Create a Kubernetes secret

Store the kubeconfig in a Kubernetes secret in the same cluster where Sensor is running:

```bash
kubectl create secret generic remote-cluster-config \
  --from-file=kubeconfig=/path/to/your/kubeconfig \
  -n stackrox
```

The secret must have a key named `kubeconfig` containing the kubeconfig data.

### 3. Configure Sensor

Set the environment variables on the Sensor deployment:

```bash
kubectl -n stackrox set env deploy/sensor \
  ROX_REMOTE_CLUSTER_SECRET=remote-cluster-config \
  ROX_REMOTE_CLUSTER_SECRET_NAMESPACE=stackrox
```

### 4. Restart Sensor

```bash
kubectl -n stackrox rollout restart deploy/sensor
```

## How It Works

1. When Sensor starts, it checks if `ROX_REMOTE_CLUSTER_SECRET` is set
2. If set, it creates a `RemoteClientManager` instance
3. The manager uses the local cluster client to read the specified secret
4. It extracts the kubeconfig from the secret's `kubeconfig` key
5. It creates a new Kubernetes client using the remote cluster configuration
6. Sensor uses this remote client for all Kubernetes operations instead of the local cluster

## Architecture

The implementation follows the same pattern as the fake workload generator:

- `NewRemoteClientManager()`: Creates a manager if the feature is enabled (returns nil otherwise)
- `InitializeRemoteClient()`: Reads the secret and creates the remote cluster client
- `Client()`: Returns the remote cluster client interface

The integration in `sensor/kubernetes/main.go` prioritizes:
1. Remote cluster (if `ROX_REMOTE_CLUSTER_SECRET` is set)
2. Fake workload generator (if workload file exists)
3. Local cluster (default)

## Important Notes

- Sensor still runs in the local cluster but monitors the remote cluster
- The local cluster client is preserved for pod ownership introspection
- The remote cluster must be accessible from the Sensor pod's network
- Ensure the kubeconfig has appropriate permissions for all resources Sensor needs to monitor
- This feature is independent of and does not interfere with the fake workload generator

## Example Secret YAML

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: remote-cluster-config
  namespace: stackrox
type: Opaque
data:
  kubeconfig: <base64-encoded-kubeconfig>
```

## Security Considerations

- The kubeconfig secret should have restricted RBAC permissions
- Use service accounts with minimal required permissions in the kubeconfig
- Consider using short-lived credentials or certificate rotation
- The secret should only be readable by the Sensor service account
