# StackRox Development Pipeline

Tekton pipeline for building and deploying StackRox with custom code changes.

## Overview

This pipeline provides fast iteration cycles for StackRox development:

1. **Clone** source from GitHub or Forgejo
2. **Build** central binary (and roxctl)
3. **Create** dev image by overlaying binaries on released base
4. **Deploy** to KinD cluster running in KubeVirt VM
5. **Validate** deployment with smoke tests

The overlay approach dramatically reduces build time compared to full image builds since it reuses the released base image (which includes UI, dependencies, etc.) and only replaces the Go binaries you're changing.

## Setup

```bash
# Create namespace
kubectl create namespace stackrox-dev

# Apply RBAC
kubectl apply -f rbac.yaml

# Apply pipeline
kubectl apply -f pipeline-dev.yaml
```

## Usage

### Basic run (from GitHub master)

```bash
kubectl create -f pipelinerun-example.yaml
```

### Run with feature flags

```bash
cat <<EOF | kubectl create -f -
apiVersion: tekton.dev/v1
kind: PipelineRun
metadata:
  generateName: stackrox-risk-plugins-
  namespace: stackrox-dev
spec:
  pipelineRef:
    name: stackrox-dev
  params:
    - name: git-revision
      value: "risk-plugins"
    - name: image-tag
      value: "risk-plugins"
    - name: feature-flags
      value: "ROX_PLUGIN_RISK_SCORING=true"
  taskRunTemplate:
    serviceAccountName: stackrox-dev-sa
  workspaces:
    - name: source
      volumeClaimTemplate:
        spec:
          accessModes: [ReadWriteOnce]
          resources:
            requests:
              storage: 10Gi
EOF
```

### Build only (no deployment)

```bash
cat <<EOF | kubectl create -f -
apiVersion: tekton.dev/v1
kind: PipelineRun
metadata:
  generateName: stackrox-build-
  namespace: stackrox-dev
spec:
  pipelineRef:
    name: stackrox-dev
  params:
    - name: skip-deploy
      value: "true"
    - name: image-tag
      value: "my-feature"
  taskRunTemplate:
    serviceAccountName: stackrox-dev-sa
  workspaces:
    - name: source
      volumeClaimTemplate:
        spec:
          accessModes: [ReadWriteOnce]
          resources:
            requests:
              storage: 10Gi
EOF
```

## Parameters

| Parameter | Default | Description |
|-----------|---------|-------------|
| `git-url` | GitHub stackrox | Source repository URL |
| `git-revision` | master | Branch, tag, or commit SHA |
| `k8s-version` | v1.29.2 | Kubernetes version for KinD |
| `vm-size` | medium | VM size (small/medium/large) |
| `image-tag` | dev | Tag for built image |
| `base-image-tag` | (auto) | Base image from quay.io/rhacs-eng/main |
| `feature-flags` | (none) | Space-separated ROX_* env vars |
| `skip-deploy` | false | Build image only, skip deployment |

## VM Sizes

| Size | CPUs | Memory |
|------|------|--------|
| small | 4 | 8 GB |
| medium | 8 | 16 GB |
| large | 16 | 32 GB |

## Monitoring

```bash
# Watch pipeline progress
tkn pipelinerun logs -f <run-name> -n stackrox-dev

# List runs
tkn pipelinerun list -n stackrox-dev
```

## Manual Deployment (Simplified)

Use `pipeline-build.yaml` for build-only, then deploy manually:

```bash
# 1. Run build pipeline
kubectl create -f - <<EOF
apiVersion: tekton.dev/v1
kind: PipelineRun
metadata:
  generateName: stackrox-build-
  namespace: stackrox-dev
spec:
  pipelineRef:
    name: stackrox-build
  params:
    - name: image-tag
      value: "dev"
  taskRunTemplate:
    serviceAccountName: stackrox-dev-sa
  workspaces:
    - name: source
      volumeClaimTemplate:
        spec:
          accessModes: [ReadWriteOnce]
          resources:
            requests:
              storage: 10Gi
EOF

# 2. Create KinD VM
kubectl apply -f kind-vm.yaml

# 3. Wait for VM and get kubeconfig
VMI_IP=$(kubectl get vmi stackrox-kind -n stackrox-dev -o jsonpath='{.status.interfaces[0].ipAddress}')
curl -s "http://${VMI_IP}:8080/kubeconfig" > /tmp/kind-kubeconfig

# 4. Build roxctl locally (needs version info)
cd /path/to/stackrox
eval "$(./status.sh | sed 's/ /=/')"
go build -ldflags="-X 'github.com/stackrox/rox/pkg/version/internal.MainVersion=${STABLE_MAIN_VERSION}'" \
  -o /tmp/roxctl ./roxctl/main.go

# 5. Deploy with dev-friendly resources
./deploy.sh /tmp/kind-kubeconfig
```

## Accessing the deployed cluster

After deployment completes, port-forward to Central:

```bash
# Get the kubeconfig from the workspace PVC (while pipeline is running)
# Or from the VM directly:
VMI_IP=$(kubectl get vmi -n stackrox-dev -l app=stackrox-dev \
  -o jsonpath='{.items[0].status.interfaces[0].ipAddress}')
curl -s "http://${VMI_IP}:8080/kubeconfig" > /tmp/kind-kubeconfig

# Port-forward
KUBECONFIG=/tmp/kind-kubeconfig kubectl port-forward -n stackrox svc/central 8443:443

# Get admin password
KUBECONFIG=/tmp/kind-kubeconfig kubectl get secret central-htpasswd -n stackrox \
  -o jsonpath='{.data.password}' | base64 -d

# Access UI
open https://localhost:8443
```

## Architecture

```
                                    ┌─────────────────────┐
                                    │ quay.io/rhacs-eng/  │
                                    │ main:BASE_TAG       │
                                    └─────────┬───────────┘
                                              │
┌──────────┐    ┌──────────┐    ┌─────────────┴───────────┐
│ Git Repo │───▶│ Build Go │───▶│ Overlay rebuilt binary  │
└──────────┘    │ binaries │    │ on base image           │
                └──────────┘    └─────────────┬───────────┘
                                              │
                                              ▼
                                 ┌─────────────────────────┐
                                 │ dev-registry            │
                                 │ stackrox/main:IMAGE_TAG │
                                 └─────────────┬───────────┘
                                               │
         ┌─────────────────────────────────────┘
         │
         ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│ KubeVirt VM     │───▶│ KinD Cluster    │───▶│ StackRox        │
│ (Podman+KinD)   │    │ (containerd)    │    │ (Central+Sensor)│
└─────────────────┘    └─────────────────┘    └─────────────────┘
```
