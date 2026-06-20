#!/usr/bin/env bash
#
# Build script called by Skaffold's custom builder.
#
# Strategy:
#   1. Get the image tag from `make tag`
#   2. Try to pull that exact tag from the remote registry (CI already built it)
#   3. If pull succeeds: tag + push to local registry. No local build needed.
#   4. If pull fails: build locally using existing Make targets.
#
# Skaffold sets $IMAGE to the target image reference (e.g., localhost:5000/main:tag).

set -euo pipefail

TAG="$(make --quiet --no-print-directory tag 2>/dev/null)"
REGISTRY="${DEFAULT_IMAGE_REGISTRY:-$(make --quiet --no-print-directory default-image-registry 2>/dev/null)}"

if command -v docker >/dev/null 2>&1; then
    RT=docker
else
    RT=podman
fi

# --- Try to pull the pre-built image from the remote registry ---

REMOTE_IMAGE="${REGISTRY}/main:${TAG}"

if [[ "$TAG" != *-dirty ]]; then
    echo "=== Trying to pull ${REMOTE_IMAGE} ==="
    if $RT pull "$REMOTE_IMAGE" 2>/dev/null; then
        echo "=== Pulled successfully — pushing to local registry ==="
        $RT tag "$REMOTE_IMAGE" "$IMAGE"
        if [[ "$RT" == "podman" ]]; then
            $RT push "$IMAGE" --tls-verify=false
        else
            $RT push "$IMAGE"
        fi
        echo "=== Done (pulled, no local build needed) ==="
        exit 0
    fi
    echo "=== Pull failed — building locally ==="
fi

# --- Local build using existing Make targets ---
# main-build-nodeps: compiles all Go binaries natively (no Docker container)
# copy-binaries-to-image-dir: stages binaries into image/rhel/bin/
# docker-build-main-image: builds the container image with COPY --link

echo "=== Building Go binaries ==="

export BUILD_TAG=0.0.0
export SHORTCOMMIT=dev
export STABLE_COLLECTOR_VERSION=0.0.0
export STABLE_FACT_VERSION=0.0.0
export STABLE_SCANNER_VERSION=0.0.0
export SKIP_UI_BUILD="${SKIP_UI_BUILD:-1}"
export CGO_ENABLED=0
export GOOS=linux

make main-build-nodeps

echo "=== Staging binaries ==="

make copy-go-binaries-to-image-dir CI=1 2>/dev/null || {
    # CI=1 path needs all-platform roxctl; fall back to manual copy
    mkdir -p image/rhel/bin
    GOARCH="$(go env GOARCH)"
    cp "bin/linux_${GOARCH}/central"            image/rhel/bin/central
    cp "bin/linux_${GOARCH}/config-controller"  image/rhel/bin/config-controller
    cp "bin/linux_${GOARCH}/migrator"           image/rhel/bin/migrator
    cp "bin/linux_${GOARCH}/kubernetes"          image/rhel/bin/kubernetes-sensor
    cp "bin/linux_${GOARCH}/upgrader"            image/rhel/bin/sensor-upgrader
    cp "bin/linux_${GOARCH}/admission-control"  image/rhel/bin/admission-control
    cp "bin/linux_${GOARCH}/compliance"          image/rhel/bin/compliance
    cp "bin/linux_${GOARCH}/roxagent"            image/rhel/bin/roxagent
    cp "bin/linux_${GOARCH}/roxctl"              "image/rhel/bin/roxctl-linux-${GOARCH}"
    find image/rhel/bin -type f -exec chmod +x {} \;
}

# Ensure UI/docs/notices dirs exist (stubs for dev if not built)
mkdir -p image/rhel/ui/build image/rhel/docs/api/v1 image/rhel/docs/api/v2 image/rhel/THIRD_PARTY_NOTICES
[[ -d ui/build ]] && cp -r ui/build image/rhel/ui/ 2>/dev/null || true
[[ -f image/rhel/docs/api/v1/swagger.json ]] || echo '{}' > image/rhel/docs/api/v1/swagger.json
[[ -f image/rhel/docs/api/v2/swagger.json ]] || echo '{}' > image/rhel/docs/api/v2/swagger.json

echo "=== Building container image ==="

TAG="$(make --quiet --no-print-directory tag 2>/dev/null)"
$RT build \
    -t "${REGISTRY}/main:${TAG}" \
    --build-arg ROX_PRODUCT_BRANDING="$(make --quiet --no-print-directory product-branding 2>/dev/null)" \
    --build-arg TARGET_ARCH="$(go env GOARCH)" \
    --build-arg ROX_IMAGE_FLAVOR="$(make --quiet --no-print-directory image-flavor 2>/dev/null)" \
    --build-arg LABEL_VERSION="${TAG}" \
    --build-arg LABEL_RELEASE="${TAG}" \
    --file image/rhel/Dockerfile \
    image/rhel

# Tag and push to local registry
$RT tag "${REGISTRY}/main:${TAG}" "$IMAGE"
if [[ "$RT" == "podman" ]]; then
    $RT push "$IMAGE" --tls-verify=false
else
    $RT push "$IMAGE"
fi
