#!/usr/bin/env bash
set -euo pipefail

# Build and deploy vsock-loadgen for OCP
# Usage: ./build-loadgen.sh [--no-push] [--no-restart]
#
# Options:
#   --no-push: Build locally only, don't push to registry
#   --no-restart: Don't restart DaemonSet after push (only if deployed)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../../.." && pwd)"

DEFAULT_USER="${USER:-developer}"
IMAGE_NAME="${VSOCK_LOADGEN_IMAGE:-quay.io/${DEFAULT_USER}/stackrox/vsock-loadgen}"
IMAGE_TAG="${VSOCK_LOADGEN_TAG:-latest}"
FULL_IMAGE="${IMAGE_NAME}:${IMAGE_TAG}"

PUSH=true
RESTART=true

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --no-push)
            PUSH=false
            shift
            ;;
        --no-restart)
            RESTART=false
            shift
            ;;
        *)
            echo "Unknown argument: $1"
            echo "Usage: $0 [--no-push] [--no-restart]"
            exit 1
            ;;
    esac
done

echo "╔═══════════════════════════════════════════════════════════════╗"
echo "║  Building vsock-loadgen                                      ║"
echo "╚═══════════════════════════════════════════════════════════════╝"
echo ""
echo "📦 Image: ${FULL_IMAGE}"
echo ""

# Step 1: Build binary locally (fast, no Docker build stage needed)
echo "🔨 Building Go binary..."
cd "${REPO_ROOT}"
if ! CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /tmp/vsock-loadgen ./compliance/virtualmachines/loadgen; then
    echo "   ✗ Go build failed"
    exit 1
fi
echo "   ✓ Binary built"
echo ""

# Step 2: Create minimal Dockerfile
# Note: DaemonSet runs as root (runAsUser: 0) for /dev/vsock access
cat > /tmp/Dockerfile.vsock-loadgen <<EOF
FROM gcr.io/distroless/static-debian12:nonroot
COPY vsock-loadgen /usr/local/bin/vsock-loadgen
ENTRYPOINT ["/usr/local/bin/vsock-loadgen"]
EOF

# Step 3: Build minimal container image
echo "📦 Building container image..."
docker build --platform linux/amd64 -f /tmp/Dockerfile.vsock-loadgen -t "${FULL_IMAGE}" /tmp
echo "   ✓ Image built"
echo ""

# Step 4: Push to registry
if [[ "${PUSH}" == "true" ]]; then
    echo "📤 Pushing to Quay..."
    docker push "${FULL_IMAGE}"
    echo "   ✓ Image pushed"
    echo ""
fi

# Step 5: Restart DaemonSet if deployed and requested
if [[ "${PUSH}" == "true" ]] && [[ "${RESTART}" == "true" ]]; then
    if kubectl -n stackrox get daemonset vsock-loadgen &> /dev/null; then
        echo "🔄 Restarting DaemonSet..."
        kubectl -n stackrox rollout restart daemonset/vsock-loadgen
        echo "   ✓ Restart initiated"
        echo ""

        echo "⏳ Waiting for rollout..."
        kubectl -n stackrox rollout status daemonset/vsock-loadgen --timeout=60s
        echo ""
    fi
fi

echo "╔═══════════════════════════════════════════════════════════════╗"
echo "║  ✅ Build complete!                                           ║"
echo "╚═══════════════════════════════════════════════════════════════╝"
echo ""

if [[ "${PUSH}" == "true" ]]; then
    echo "📊 Image: ${FULL_IMAGE}"
    if kubectl -n stackrox get daemonset vsock-loadgen &> /dev/null 2>&1; then
        echo ""
        echo "Running pods:"
        kubectl -n stackrox get pods -l app=vsock-loadgen -o wide
    else
        echo ""
        echo "💡 Deploy with: ./run-loadgen.sh"
    fi
else
    echo "💡 Image built locally (not pushed)"
    echo "   Run without --no-push to push to registry"
fi
echo ""

# Cleanup
rm -f /tmp/vsock-loadgen /tmp/Dockerfile.vsock-loadgen
