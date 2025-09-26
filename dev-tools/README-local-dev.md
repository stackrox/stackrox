# Fast Local StackRox Development with Tekton

This directory contains tools for fast local StackRox development using Tekton pipelines and the custom `local-dev` image flavor.

## Quick Start

```bash
# Build StackRox images locally
./dev-tools/local-build.sh

# Deploy with custom images
export ROX_IMAGE_FLAVOR=local-dev
export ROX_LOCAL_REGISTRY=localhost:5000
export ROX_LOCAL_TAG=latest
cd ./installer
go build -o bin/installer ./installer
./bin/installer apply central
```

## Features

- **🚀 Fast builds**: 5-task pipeline vs 13-task original (simplified for speed)
- **📦 Local registry**: Pushes to localhost:5000 by default
- **🔄 Aggressive caching**: S3/MinIO caching for Go modules and build cache
- **🎯 Main image focus**: Builds only main StackRox image (can expand later)
- **🧰 Developer-friendly**: Single command with smart defaults
- **🔗 Helm integration**: Automatic integration via `local-dev` flavor

## Architecture

### Components

1. **Custom Image Flavor** (`pkg/images/defaults/`)
   - `local-dev` flavor with environment variable configuration
   - Defaults to `localhost:5000` registry and `latest` tag
   - Configurable via `ROX_LOCAL_REGISTRY` and `ROX_LOCAL_TAG`

2. **Tekton Pipeline** (`dev-tools/tekton/`)
   - Streamlined 5-task pipeline for fast iteration
   - S3-compatible caching with MinIO backend
   - Official StackRox builder images
   - Git commit + custom tag support

3. **Wrapper Script** (`dev-tools/local-build.sh`)
   - Easy-to-use interface with smart defaults
   - Handles Tekton setup and pipeline execution
   - Automatic Helm integration setup
   - Optional StackRox deployment

### Performance Optimizations

- **Skip scanner v2 compilation**: Saves 5-10 minutes
- **Skip documentation generation**: Saves 2-3 minutes
- **Go module caching**: Saves 3-5 minutes on subsequent builds
- **Go build caching**: Saves 2-4 minutes on incremental builds
- **Reduced workspace**: 20Gi vs 40Gi (sufficient for main image)

## Usage

### Basic Build

```bash
# Build with defaults (localhost:5000/stackrox/main:latest)
./dev-tools/local-build.sh
```

### Custom Configuration

```bash
# Custom registry and tag
./dev-tools/local-build.sh --registry my-registry:5000 --tag v1.0.0

# Build from specific branch
./dev-tools/local-build.sh --revision feature-branch

# Build and deploy automatically
./dev-tools/local-build.sh --deploy

# Use custom builder image
./dev-tools/local-build.sh --builder-image my-registry/builder:latest
```

### Environment Variables

```bash
# Set default registry and tag
export ROX_LOCAL_REGISTRY=my-registry:5000
export ROX_LOCAL_TAG=v1.0.0

# Use local-dev flavor for deployment
export ROX_IMAGE_FLAVOR=local-dev

# Custom Tekton namespace
export TEKTON_NAMESPACE=my-builds
```

### Manual Pipeline Execution

```bash
# Apply Tekton resources
kubectl apply -f dev-tools/tekton/

# Create PipelineRun
kubectl apply -f dev-tools/tekton/pipelinerun-local-dev.yaml

# Monitor progress
kubectl logs -f pipelinerun/stackrox-local-dev-xxxxx -n stackrox-builds
```

## Prerequisites

### Required Tools

- `kubectl` - Kubernetes cluster access
- `git` - Source code management
- Tekton Pipelines installed in cluster

### Cluster Requirements

- Kubernetes cluster with Tekton Pipelines
- 20Gi+ storage for build workspace
- Container registry access (localhost:5000 for local development)

### Optional Components

- MinIO for S3-compatible caching (auto-installed by script)
- Local container registry (kind-registry for kind clusters)

## Pipeline Details

### Task Sequence

1. **setup-aws-credentials** - Configure MinIO credentials for caching
2. **fetch-source** - Clone git repository with custom task
3. **get-git-commit** - Extract commit SHA for image tagging
4. **build-go-binaries** - Compile StackRox binaries with caching
5. **build-image** - Create container image with buildah
6. **tag-image** - Tag with both commit SHA and custom tag

### Caching Strategy

- **Go modules**: Cached by go.mod file content hash
- **Go build cache**: Cached by source file patterns
- **Cache storage**: MinIO S3-compatible object storage
- **Cache invalidation**: Automatic based on dependency changes

### Build Outputs

- `{registry}/stackrox/main:{git-commit}` - Git commit tagged image
- `{registry}/stackrox/main:{custom-tag}` - Custom tagged image

## Integration with StackRox Deployment

### Using the Installer

```bash
# Set up environment
export ROX_IMAGE_FLAVOR=local-dev
export ROX_LOCAL_REGISTRY=localhost:5000
export ROX_LOCAL_TAG=latest

# Build and deploy
cd ./installer
go build -o bin/installer ./installer
./bin/installer apply central
./bin/installer apply crs
./bin/installer apply securedcluster
```

### Using Helm Charts

```bash
# The local-dev flavor automatically configures Helm charts
export ROX_IMAGE_FLAVOR=local-dev
export ROX_LOCAL_REGISTRY=localhost:5000
export ROX_LOCAL_TAG=latest

# Generate Helm charts with custom images
roxctl central generate k8s --image-defaults=local-dev
```

### Using the Operator

```bash
# Set environment variables for operator
export RELATED_IMAGE_MAIN=localhost:5000/main:latest
export RELATED_IMAGE_CENTRAL_DB=localhost:5000/central-db:latest

# Deploy via operator CRDs
kubectl apply -f central-crd.yaml
```

## Troubleshooting

### Common Issues

**Build fails with "No space left on device"**
- Increase workspace size in `pipelinerun-local-dev.yaml`
- Clean up old PipelineRuns: `kubectl delete pipelinerun --all -n stackrox-builds`

**MinIO connection fails**
- Check MinIO pod status: `kubectl get pods -l app=minio`
- Restart MinIO: `kubectl rollout restart deployment/minio`
- Check service: `kubectl get svc minio`

**Image push fails to localhost:5000**
- Verify registry is running: `docker ps | grep registry`
- Check buildah TLS settings in pipeline (should be `--tls-verify=false`)
- For kind clusters: ensure registry is properly configured

**Git clone fails**
- Check repository access
- For private repos, configure git credentials in cluster
- Verify builder image has git installed

### Debugging Commands

```bash
# Check pipeline status
kubectl get pipelinerun -n stackrox-builds

# View pipeline logs
kubectl logs pipelinerun/stackrox-local-dev-xxxxx -n stackrox-builds

# Debug specific task
kubectl describe taskrun/build-go-binaries-xxxxx -n stackrox-builds

# Check cache status
kubectl exec -it deployment/minio -- mc ls local/local-dev-cache

# Verify built images
curl -X GET http://localhost:5000/v2/_catalog
curl -X GET http://localhost:5000/v2/stackrox/main/tags/list
```

### Performance Tuning

**Improve cache hit rates**
- Avoid changing go.mod files unnecessarily
- Keep builder image consistent
- Use dedicated cache bucket per project

**Reduce build time**
- Use faster storage class for workspace PVC
- Increase CPU/memory limits for build tasks
- Use local registry to reduce push time

**Optimize for CI/CD**
- Use shared cache bucket across team
- Pre-warm cache with common dependencies
- Use consistent builder image versions

## Development Workflow

### Typical Development Cycle

1. **Make code changes** in your local repository
2. **Build custom image**: `./dev-tools/local-build.sh`
3. **Deploy for testing**: `./dev-tools/local-build.sh --deploy`
4. **Iterate**: Make changes and rebuild (faster with cache)
5. **Integration testing**: Deploy full cluster with custom images

### Team Collaboration

```bash
# Share custom images via team registry
./dev-tools/local-build.sh --registry team-registry:5000 --tag feature-xyz

# Use shared cache bucket
./dev-tools/local-build.sh --cache-bucket team-dev-cache

# Standardize on builder image
./dev-tools/local-build.sh --builder-image team-registry/stackrox-builder:v1.0.0
```

### CI/CD Integration

The local development tools can be adapted for CI/CD:

1. **Use in CI pipelines** for feature branch testing
2. **Shared cache** for faster CI builds
3. **Multi-arch builds** by extending pipeline
4. **Automated testing** with custom images

## Extending the Pipeline

### Adding Components

To add scanner v2 or other components:

1. Add tasks to `pipeline-local-dev.yaml`
2. Extend `build-go-binaries` task with additional make targets
3. Update image tagging in final steps

### Multi-Architecture Support

1. Extend buildah task with `--platform` parameter
2. Add manifest list creation step
3. Update cache patterns for architecture-specific builds

### Custom Builder Images

1. Create builder image with required tools
2. Test with: `./dev-tools/local-build.sh --builder-image my-image`
3. Update default in pipeline once validated

## Support

For issues with the local development tools:

1. Check the troubleshooting section above
2. Review pipeline logs for specific error messages
3. Verify prerequisites and cluster configuration
4. Test with minimal configuration first

The tools are designed to be self-contained and work with standard Kubernetes + Tekton clusters.