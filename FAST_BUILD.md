# Fast Inner Loop Development for StackRox

This document describes the fast build workflow for rapid iteration during StackRox development.

## Overview

The fast build workflow significantly speeds up the inner loop by:

1. Using a pre-built base image from `quay.io/rhacs-eng/main` instead of building from scratch
2. Only building and copying the Go binaries you're actively developing
3. Building static binaries with `CGO_ENABLED=0` to avoid GLIBC version compatibility issues
4. Skipping UI builds, RPM downloads, and other time-consuming steps

**Traditional build time**: 15-30 minutes
**Fast build time**: 2-5 minutes

## Key Technical Details

### Static Binary Compilation

All binaries are built with `CGO_ENABLED=0` to create static binaries without GLIBC dependencies. This solves compatibility issues between:

* **Base image**: GLIBC 2.28 (from 2018)
* **Development container**: GLIBC 2.42 (2025)

Static binaries are portable across different GLIBC versions and run reliably in the older base image.

### Dockerfile Design

The `Dockerfile.fastbuild` uses a two-stage copy approach for efficiency:

1. **Copy central binary** directly to `/stackrox/central`
2. **Copy service binaries** to a temporary location, then rename and move to `/stackrox/bin/`

This design:
* Minimizes the number of COPY layers
* Ensures correct binary naming (e.g., `kubernetes` → `kubernetes-sensor`)
* Sets proper ownership (UID 4000:4000) for security
* Cleans up temporary files to reduce image size

The Dockerfile uses `bin/linux_${GOARCH}/` as the build context instead of the repository root, which works around the `.containerignore` file that filters out the `/bin/` directory.

## Quick Start

### Using Session Helper Scripts (Recommended)

The session includes helper scripts that automate the entire workflow:

```bash
# Build binaries, create image, and load into kind cluster
~/workspace/sessions/image-build/build.sh

# Deploy to the cluster using roxie
~/workspace/sessions/image-build/deploy.sh

# Validate the deployment
~/workspace/sessions/image-build/validate.sh
```

### Using Make Targets Directly

```bash
cd ~/workspace/sessions/image-build/stackrox

# Build just the binaries
make fast-binaries

# Build binaries and create Docker image
make fast-image

# Build, create image, and push to registry
make fast-push-registry

# Complete inner loop (recommended)
make fast-inner-loop
```

## Customization

### Using a Different Base Image Tag

By default, the workflow uses `quay.io/rhacs-eng/main:latest`. You can customize this:

```bash
# Using environment variable
BASE_TAG=4.6.x-latest ~/workspace/sessions/image-build/build.sh

# Using make parameter
make fast-inner-loop BASE_TAG=4.6.x-latest
```

### Using a Different Image Tag

By default, the built image is tagged as `stackrox/main:local-dev`. You can customize this:

```bash
# Using environment variable
IMAGE_TAG=my-feature ~/workspace/sessions/image-build/build.sh

# Using make parameter
make fast-inner-loop IMAGE_TAG=my-feature
```

### Using a Different Cluster

If you're not using the default `stackrox-image-build` cluster:

```bash
make fast-inner-loop CLUSTER_NAME=my-cluster
```

## What Gets Built

The fast build creates these binaries using `go-build.sh`:

* `central` - Main central service
* `migrator` - Database migration tool
* `compliance` - Compliance checking service
* `kubernetes-sensor` - Kubernetes sensor for monitoring
* `sensor-upgrader` - Sensor upgrade utility
* `admission-control` - Admission control webhook
* `config-controller` - Configuration management controller
* `init-tls-certs` - TLS certificate initialization

All binaries are built with the same flags as the full build using `scripts/go-build.sh`.

## Files Created

* `Dockerfile.fastbuild` - Dockerfile that layers local binaries over the base image
* `Makefile` - Added fast build targets at the end:
  * `fast-binaries` - Build Go binaries only
  * `fast-image` - Build Docker image with local binaries
  * `fast-push-registry` - Push image to registry
  * `fast-inner-loop` - Complete workflow

## How It Works

### Step 1: Build Binaries

```bash
make fast-binaries
```

This runs `go-build.sh` to compile the Go binaries with the correct flags:
* Builds with `CGO_ENABLED=0` for static binaries
* Builds with proper ldflags from `status.sh`
* Supports `DEBUG_BUILD=yes` for debugging
* Uses `GOTAGS` for conditional compilation
* Outputs to `bin/linux_${GOARCH}/` directory (architecture-aware)

### Step 2: Create Docker Image

```bash
make fast-image
```

This uses `Dockerfile.fastbuild` to:
1. Pull the base image from `quay.io/rhacs-eng/main:${BASE_TAG}`
2. Copy locally-built static binaries over the ones in the base image:
   * `central` → `/stackrox/central`
   * Other binaries → `/stackrox/bin/` (migrator, compliance, kubernetes-sensor, etc.)
3. Set correct ownership (UID 4000:4000)
4. Tag as `stackrox/main:${IMAGE_TAG}`

The Dockerfile uses `bin/linux_${GOARCH}/` as the build context to work around `.containerignore` filtering.

### Step 3: Push to Registry

```bash
make fast-push-registry
```

This tags and pushes the image to the kind registry at `localhost:5001`, making it available to the cluster as `kind-registry:5000/stackrox/main:local-dev`.

## Debugging

### Enable Debug Build

To build with debug symbols and disable optimizations:

```bash
DEBUG_BUILD=yes make fast-inner-loop
```

### Enable Race Detection

To build with Go's race detector:

```bash
RACE=true make fast-binaries
```

**Note**: Race detection requires `CGO_ENABLED=1`, which conflicts with the static binary approach. Using race detection will:
* Enable dynamic linking (losing GLIBC compatibility benefits)
* Slow down the build significantly
* Require matching GLIBC versions between dev environment and base image

### Check Build Flags

The `go-build.sh` script automatically sets build flags based on `status.sh`. To see what flags are being used, check the build output.

## Troubleshooting

### Base image pull fails

If you can't pull from `quay.io/rhacs-eng`, ensure you're authenticated:

```bash
docker login quay.io
```

### Binaries not found

Ensure you run `make fast-binaries` or `make fast-image` before trying to build the Docker image directly. The Makefile dependencies handle this automatically.

### GLIBC version errors

If you see errors like `GLIBC_X.XX not found`, it means CGO is enabled. Ensure:

```bash
# Check that CGO is disabled in the Makefile
grep "CGO_ENABLED=0" Makefile

# Verify binaries are static
ldd bin/linux_*/central  # Should output "not a dynamic executable"
```

### Registry push fails

Check that the kind registry is running:

```bash
docker ps | grep registry
```

Ensure the registry is accessible at `localhost:5001`:

```bash
curl localhost:5001/v2/
```

### Deployment fails with image pull errors

When deploying, ensure:

1. Image is in the registry: `curl -X GET localhost:5001/v2/stackrox/main/tags/list`
2. Image name in manifests is `kind-registry:5000/stackrox/main:local-dev`
3. `imagePullPolicy` is set to `IfNotPresent`

## Complete Workflow Example

```bash
# 1. Make your code changes
vim central/somefeature/feature.go

# 2. Build and load into cluster
~/workspace/sessions/image-build/build.sh

# 3. Deploy (or redeploy) to cluster
~/workspace/sessions/image-build/deploy.sh

# 4. Test your changes
kubectl logs -n stackrox deploy/central -f

# 5. Iterate - repeat steps 1-4
```

## Deployment Options

### Option 1: Using roxctl (Recommended)

Generate and deploy using roxctl-generated manifests:

```bash
# Generate manifests
roxctl central generate k8s pvc --output-dir /tmp/central-bundle

# Create registry pull secret
kubectl create secret generic stackrox \
  --from-file=.dockerconfigjson=~/.docker/config.json \
  --type=kubernetes.io/dockerconfigjson

# Update image references in manifests (if needed)
find /tmp/central-bundle -name "*.yaml" -exec sed -i 's|docker.io/localhost/||g' {} \;

# Deploy
./central/scripts/setup.sh
kubectl create -R -f central/
```

### Option 2: Using roxie

The deploy script can use roxie with these key parameters:

* `--main-image stackrox/main:local-dev` - Use the locally-built image
* `--image-pull-policy Never` - Don't pull from registry, use local image

**Note**: The operator deployment may have compatibility issues. Use roxctl (Option 1) for more reliable deployments.

## Performance Comparison

| Build Type | Time | What It Builds |
|------------|------|----------------|
| Full build (`make image`) | 15-30 min | Everything: UI, binaries, RPMs, base images |
| Fast build (`make fast-inner-loop`) | 2-5 min | Just the Go binaries |

## When to Use Full Build

Use the full build when:

* You're changing UI code
* You're changing the base image or Dockerfile
* You're adding new RPM dependencies
* You're preparing for a release or PR
* You need to test the exact production image

## When to Use Fast Build

Use the fast build when:

* You're iterating on Go code in central or sensor components
* You want rapid feedback during development
* You're debugging or testing specific features
* You're doing inner loop development

## Additional Resources
