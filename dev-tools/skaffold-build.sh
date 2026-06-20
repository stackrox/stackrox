#!/usr/bin/env bash
#
# Build script called by Skaffold's custom builder.
#
# Strategy:
#   1. Get the image tag from `make tag` (e.g., 4.12.x-254-g3f20103e67)
#   2. Try to pull that exact tag from the remote registry (CI already built it)
#   3. If pull succeeds: tag as $IMAGE + push to local registry. Done in ~30s.
#   4. If pull fails (dirty tree, unpushed commit): build locally using the
#      main Dockerfile with COPY --link (only changed binary layers rebuild).
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
        echo "=== Pulled successfully — tagging as ${IMAGE} ==="
        $RT tag "$REMOTE_IMAGE" "$IMAGE"
        if [[ "$RT" == "podman" ]]; then
            $RT push "$IMAGE" --tls-verify=false
        else
            $RT push "$IMAGE"
        fi
        echo "=== Done (pulled, no local build needed) ==="
        exit 0
    fi
    echo "=== Pull failed — falling back to local build ==="
fi

# --- Local build: cross-compile + stage into image/rhel/ + docker build ---

echo "=== Cross-compiling Go binaries (GOOS=linux GOARCH=${GOARCH} CGO_ENABLED=0) ==="

BINS=(
    "central:./central"
    "migrator:./migrator"
    "compliance:./compliance/cmd/compliance"
    "kubernetes-sensor:./sensor/kubernetes"
    "sensor-upgrader:./sensor/upgrader"
    "admission-control:./sensor/admission-control"
    "config-controller:./config-controller"
    "roxagent:./compliance/virtualmachines/roxagent"
)

VERSION_PKG="github.com/stackrox/rox/pkg/version/internal"
LDFLAGS="-s -w \
 -X ${VERSION_PKG}.MainVersion=0.0.0-dev \
 -X ${VERSION_PKG}.CollectorVersion=0.0.0-dev \
 -X ${VERSION_PKG}.ScannerVersion=0.0.0-dev \
 -X ${VERSION_PKG}.GitShortSha=dev"

for entry in "${BINS[@]}"; do
    bin="${entry%%:*}"
    pkg="${entry##*:}"
    GOOS=linux GOARCH="$GOARCH" CGO_ENABLED=0 \
        go build -buildvcs=false -trimpath -ldflags="$LDFLAGS" \
        -o "${OUTDIR}/${bin}" "$pkg"
done

# Build roxctl for the current platform
GOOS=linux GOARCH="$GOARCH" CGO_ENABLED=0 \
    go build -buildvcs=false -trimpath -ldflags="$LDFLAGS" \
    -o "${OUTDIR}/roxctl" ./roxctl

echo "=== Staging binaries into image/rhel/bin/ ==="

mkdir -p image/rhel/bin
cp "${OUTDIR}/central" image/rhel/bin/central
cp "${OUTDIR}/config-controller" image/rhel/bin/config-controller
cp "${OUTDIR}/migrator" image/rhel/bin/migrator
cp "${OUTDIR}/compliance" image/rhel/bin/compliance
cp "${OUTDIR}/kubernetes-sensor" image/rhel/bin/kubernetes-sensor
cp "${OUTDIR}/sensor-upgrader" image/rhel/bin/sensor-upgrader
cp "${OUTDIR}/admission-control" image/rhel/bin/admission-control
cp "${OUTDIR}/roxagent" image/rhel/bin/roxagent
cp "${OUTDIR}/roxctl" "image/rhel/bin/roxctl-linux-${GOARCH}"

# Ensure UI and docs exist (stubs if not built)
mkdir -p image/rhel/ui/build
[[ -f image/rhel/ui/build/index.html ]] || echo '<html><body>dev</body></html>' > image/rhel/ui/build/index.html
mkdir -p image/rhel/docs/api/v1 image/rhel/docs/api/v2
[[ -f image/rhel/docs/api/v1/swagger.json ]] || echo '{}' > image/rhel/docs/api/v1/swagger.json
[[ -f image/rhel/docs/api/v2/swagger.json ]] || echo '{}' > image/rhel/docs/api/v2/swagger.json
mkdir -p image/rhel/THIRD_PARTY_NOTICES
[[ -f image/rhel/THIRD_PARTY_NOTICES/index.html ]] || echo 'dev' > image/rhel/THIRD_PARTY_NOTICES/index.html

echo "=== Building image (COPY --link — only changed binary layers rebuild) ==="

$RT build -f image/rhel/Dockerfile \
    -t "$IMAGE" image/rhel/

echo "=== Pushing to local registry ==="

if [[ "$RT" == "podman" ]]; then
    $RT push "$IMAGE" --tls-verify=false
else
    $RT push "$IMAGE"
fi
