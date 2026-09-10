#!/usr/bin/env bash
#
# Build script called by Skaffold's custom builder.
#
# Strategy:
#   1. Get the image tag from `make tag`
#   2. Try to pull that exact tag from the remote registry (CI already built it)
#   3. If pull succeeds: tag + push to local registry. No local build needed.
#   4. If pull fails: compile locally with direct `go build` (no Make/status.sh
#      overhead), build image with BuildKit (COPY --link caching + direct push).
#
# Skaffold sets $IMAGE to the target image reference (e.g., localhost:5000/main:tag).

set -euo pipefail

GOARCH="$(go env GOARCH)"
OUTDIR="bin/linux_${GOARCH}"
REGISTRY="${DEFAULT_IMAGE_REGISTRY:-$(make --quiet --no-print-directory default-image-registry 2>/dev/null)}"
TAG="$(make --quiet --no-print-directory tag 2>/dev/null)"

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

# --- Local build: direct go build + BuildKit image ---

echo "=== Cross-compiling Go binaries ==="

BINS=(
    "central:./central"
    "migrator:./migrator"
    "compliance:./compliance/cmd/compliance"
    "kubernetes-sensor:./sensor/kubernetes"
    "sensor-upgrader:./sensor/upgrader"
    "admission-control:./sensor/admission-control"
    "config-controller:./config-controller"
    "roxagent:./compliance/virtualmachines/roxagent"
    "roxctl:./roxctl"
)

VERSION_PKG="github.com/stackrox/rox/pkg/version/internal"
GCFLAGS=""

if [[ "${DEBUG_BUILD:-}" == "yes" ]]; then
    echo "=== DEBUG BUILD: symbols preserved, optimizations disabled ==="
    LDFLAGS=" \
     -X ${VERSION_PKG}.MainVersion=0.0.0-dev \
     -X ${VERSION_PKG}.CollectorVersion=0.0.0-dev \
     -X ${VERSION_PKG}.ScannerVersion=0.0.0-dev \
     -X ${VERSION_PKG}.GitShortSha=dev"
    GCFLAGS="-gcflags=all=-N -l"
else
    LDFLAGS="-s -w \
     -X ${VERSION_PKG}.MainVersion=0.0.0-dev \
     -X ${VERSION_PKG}.CollectorVersion=0.0.0-dev \
     -X ${VERSION_PKG}.ScannerVersion=0.0.0-dev \
     -X ${VERSION_PKG}.GitShortSha=dev"
fi

for entry in "${BINS[@]}"; do
    bin="${entry%%:*}"
    pkg="${entry##*:}"
    GOOS=linux GOARCH="$GOARCH" CGO_ENABLED=0 \
        go build -buildvcs=false -trimpath $GCFLAGS -ldflags="$LDFLAGS" \
        -o "${OUTDIR}/${bin}" "$pkg"
done

echo "=== Staging binaries ==="

mkdir -p image/rhel/bin
cp "${OUTDIR}/central"            image/rhel/bin/central
cp "${OUTDIR}/config-controller"  image/rhel/bin/config-controller
cp "${OUTDIR}/migrator"           image/rhel/bin/migrator
cp "${OUTDIR}/kubernetes-sensor"  image/rhel/bin/kubernetes-sensor
cp "${OUTDIR}/sensor-upgrader"    image/rhel/bin/sensor-upgrader
cp "${OUTDIR}/admission-control"  image/rhel/bin/admission-control
cp "${OUTDIR}/compliance"         image/rhel/bin/compliance
cp "${OUTDIR}/roxagent"           image/rhel/bin/roxagent
cp "${OUTDIR}/roxctl"             "image/rhel/bin/roxctl-linux-${GOARCH}"

# Install Delve for debug builds
if [[ "${DEBUG_BUILD:-}" == "yes" ]]; then
    if [[ ! -f "$HOME/.go/bin/linux_${GOARCH}/dlv" ]]; then
        echo "    Building Delve..."
        GOOS=linux GOARCH="$GOARCH" CGO_ENABLED=0 go install github.com/go-delve/delve/cmd/dlv@latest 2>&1
    fi
    cp "$HOME/.go/bin/linux_${GOARCH}/dlv" image/rhel/bin/dlv
fi
touch image/rhel/bin/dlv-placeholder
chmod +x image/rhel/bin/*

# Ensure UI/docs/notices dirs exist (stubs for dev)
mkdir -p image/rhel/ui/build image/rhel/docs/api/v1 image/rhel/docs/api/v2 image/rhel/THIRD_PARTY_NOTICES
[[ -d ui/build ]] && cp -r ui/build image/rhel/ui/ 2>/dev/null || true
[[ -f image/rhel/docs/api/v1/swagger.json ]] || echo '{}' > image/rhel/docs/api/v1/swagger.json
[[ -f image/rhel/docs/api/v2/swagger.json ]] || echo '{}' > image/rhel/docs/api/v2/swagger.json

echo "=== Building + pushing image ==="

# Use BuildKit if available (COPY --link caching + direct push, ~2-7s)
# Fall back to podman/docker build + push (~18s)
BUILDKITD_CONTAINER="${BUILDKITD_CONTAINER:-buildkitd}"
if $RT inspect --format '{{.State.Running}}' "$BUILDKITD_CONTAINER" 2>/dev/null | grep -q true; then
    # BuildKit running in a container with image/rhel/ mounted — no context transfer
    REG_IMAGE="${IMAGE/localhost/$(echo "$BUILDKITD_CONTAINER" | sed 's/buildkitd//')registry}"
    # Convert localhost:PORT to CLUSTER-registry:5000 for in-network push
    REG_IMAGE="$(echo "$IMAGE" | sed "s|localhost:[0-9]*|${KIND_CLUSTER_NAME:-stackrox-dev}-registry:5000|")"
    $RT exec "$BUILDKITD_CONTAINER" buildctl build \
        --addr unix:///run/buildkit/buildkitd.sock \
        --frontend dockerfile.v0 \
        --local context=/context \
        --local dockerfile=/context \
        --opt "build-arg:TARGET_ARCH=${GOARCH}" \
        --opt "build-arg:DEBUG_BUILD=${DEBUG_BUILD:-no}" \
        --output "type=image,name=${REG_IMAGE},push=true,registry.insecure=true"
elif [[ -n "${BUILDKIT_HOST:-}" ]]; then
    buildctl build \
        --frontend dockerfile.v0 \
        --local context=image/rhel \
        --local dockerfile=image/rhel \
        --opt "build-arg:TARGET_ARCH=${GOARCH}" \
        --opt "build-arg:DEBUG_BUILD=${DEBUG_BUILD:-no}" \
        --output "type=image,name=${IMAGE},push=true,registry.insecure=true"
else
    $RT build \
        -t "$IMAGE" \
        --build-arg "TARGET_ARCH=${GOARCH}" \
        --build-arg "DEBUG_BUILD=${DEBUG_BUILD:-no}" \
        --file image/rhel/Dockerfile \
        image/rhel

    if [[ "$RT" == "podman" ]]; then
        $RT push "$IMAGE" --tls-verify=false
    else
        $RT push "$IMAGE"
    fi
fi
