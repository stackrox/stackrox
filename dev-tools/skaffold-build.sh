#!/usr/bin/env bash
#
# Build script called by Skaffold's custom builder.
#
# Strategy:
#   1. Get the image tag from `make tag`
#   2. Try to pull that exact tag from the remote registry (CI already built it)
#   3. If pull succeeds: tag + push to local registry. No local build needed.
#   4. If pull fails: build locally using existing Make targets (same as CI).
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

echo "=== Building (make main-build + docker-build-main-image) ==="

export SKIP_UI_BUILD="${SKIP_UI_BUILD:-1}"
export BUILD_TAG=0.0.0
export SHORTCOMMIT=dev
export GOTAGS="${GOTAGS:-}"

make main-build
make docker-build-main-image

# Tag and push to local registry
$RT tag "${REGISTRY}/main:${TAG}" "$IMAGE"
if [[ "$RT" == "podman" ]]; then
    $RT push "$IMAGE" --tls-verify=false
else
    $RT push "$IMAGE"
fi
