# VM Load Generator Scripts

Helper scripts for building and deploying the vsock load generator.

## Scripts

### `build-loadgen.sh`

Builds the load generator binary and container image, then pushes to a registry.

**Usage:**
```bash
./build-loadgen.sh [OPTIONS]
```

**Options:**
- `--no-push`: Build locally only, don't push to registry
- `--no-restart`: Don't restart DaemonSet after push (only if deployed)

**Environment Variables:**
- `VSOCK_LOADGEN_IMAGE`: Image repository (default: `quay.io/${USER}/stackrox/vsock-loadgen`)
- `VSOCK_LOADGEN_TAG`: Image tag (default: `latest`)

**Examples:**
```bash
# Full build, push, and restart
./build-loadgen.sh

# Build locally without pushing
./build-loadgen.sh --no-push

# Use custom image repository
export VSOCK_LOADGEN_IMAGE="quay.io/myorg/vsock-loadgen"
export VSOCK_LOADGEN_TAG="v1.0"
./build-loadgen.sh

# Build and push without restarting DaemonSet
./build-loadgen.sh --no-restart
```

**Build Process:**
1. Compiles Go binary for linux/amd64 (static, no CGO)
2. Creates minimal distroless-based container image
3. Pushes to registry (if `--no-push` not specified)
4. Restarts DaemonSet if deployed (if `--no-restart` not specified)

### `run-loadgen.sh`

Deploys the load generator DaemonSet to the cluster with the specified configuration.

**Usage:**
```bash
./run-loadgen.sh [CONFIG_FILE]
```

**Arguments:**
- `CONFIG_FILE`: Path to load generator config (default: `../deploy/loadgen-config.yaml`)

**Environment Variables:**
- `VSOCK_LOADGEN_IMAGE`: Override image repository
- `VSOCK_LOADGEN_TAG`: Override image tag (requires manifest update)

**Examples:**
```bash
# Deploy with default config
./run-loadgen.sh

# Deploy with custom config
./run-loadgen.sh /path/to/custom-config.yaml

# Use custom image (requires manifest update)
export VSOCK_LOADGEN_IMAGE="quay.io/myorg/vsock-loadgen"
./run-loadgen.sh
```

**Deployment Process:**
1. Validates config file exists
2. Creates/updates ConfigMap with config
3. Deploys DaemonSet manifest
4. Waits for pods to be ready
5. Shows deployment status and helpful commands

## Typical Workflow

### Initial Setup

1. **Build and push image:**
   ```bash
   ./build-loadgen.sh
   ```

2. **Edit configuration** (edit `../deploy/loadgen-config.yaml`):
   ```yaml
   loadgen:
     vmCount: 1000
     reportInterval: 60s
     payloadSize: small
   ```

3. **Deploy to cluster:**
   ```bash
   ./run-loadgen.sh
   ```

### Development Iteration

When making code changes:

```bash
# Make changes to ../main.go
# ...

# Rebuild and deploy
./build-loadgen.sh

# Check logs
kubectl -n stackrox logs -f -l app=vsock-loadgen --max-log-requests=5
```

### Configuration Changes

When changing load test parameters:

```bash
# Edit ../deploy/loadgen-config.yaml
# ...

# Redeploy with new config
./run-loadgen.sh

# Or manually update:
kubectl -n stackrox create configmap vsock-loadgen-config \
  --from-file=config.yaml=../deploy/loadgen-config.yaml \
  --dry-run=client -o yaml | kubectl apply -f -
kubectl -n stackrox rollout restart daemonset/vsock-loadgen
```

## Requirements

### Build Requirements

- Go 1.24+ installed
- Docker or Podman
- Write access to container registry
- Repository root at `../../../../` from script location

### Deployment Requirements

- kubectl configured with cluster access
- StackRox namespace (`stackrox`) exists
- Sufficient permissions to create:
  - ServiceAccounts, Roles, RoleBindings
  - ClusterRoles, ClusterRoleBindings
  - ConfigMaps, DaemonSets
- Worker nodes with:
  - `/dev/vsock` device available
  - Privileged pod support (or SCC on OpenShift)

## Troubleshooting

### Build Failures

```bash
# Check Go version
go version  # Should be 1.24+

# Verify repository structure
ls ../../../../compliance/virtualmachines/loadgen/main.go

# Check Docker/Podman
docker version
```

### Push Failures

```bash
# Login to registry
docker login quay.io

# Use custom registry you have access to
export VSOCK_LOADGEN_IMAGE="docker.io/myuser/vsock-loadgen"
./build-loadgen.sh
```

### Deployment Failures

```bash
# Check cluster connectivity
kubectl cluster-info

# Check namespace exists
kubectl get namespace stackrox

# Check RBAC permissions
kubectl auth can-i create serviceaccounts -n stackrox
kubectl auth can-i create clusterroles
kubectl auth can-i create clusterrolebindings

# View deployment events
kubectl -n stackrox get events --sort-by='.lastTimestamp'
```

### DaemonSet Not Running

```bash
# Check node affinity
kubectl -n stackrox describe daemonset vsock-loadgen

# Check worker nodes
kubectl get nodes -l '!node-role.kubernetes.io/control-plane,!node-role.kubernetes.io/master'

# Check pod events
kubectl -n stackrox describe pod -l app=vsock-loadgen
```

## See Also

- [Main README](../README.md) - Load generator overview
- [Deploy README](../deploy/README.md) - Kubernetes manifests
- [StackRox Makefile](../../../../Makefile) - Build integration
