#!/bin/bash
# deploy.sh — Deploy StackRox to KinD cluster with dev-friendly resources
#
# Usage:
#   ./deploy.sh [kubeconfig]
#
# Prerequisites:
#   - KinD cluster running (use kind-vm.yaml)
#   - roxctl built with version info (from pipeline or local build)

set -euo pipefail

KUBECONFIG="${1:-/tmp/kind-kubeconfig}"
export KUBECONFIG

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROXCTL="${ROXCTL:-/tmp/roxctl}"

# Image tags - use 4.10.0 images (DB seq 220)
# NOTE: stackrox-io "latest" is stuck at 4.5.x (seq 205), so we use explicit version
# To use nightly builds: IMAGE_TAG=4.11.x-nightly-YYYYMMDD REGISTRY=quay.io/rhacs-eng ./deploy.sh
IMAGE_TAG="${IMAGE_TAG:-4.10.0}"
REGISTRY="${REGISTRY:-quay.io/stackrox-io}"
MAIN_IMAGE="${REGISTRY}/main:${IMAGE_TAG}"
CENTRAL_DB_IMAGE="${REGISTRY}/central-db:${IMAGE_TAG}"
SCANNER_IMAGE="${REGISTRY}/scanner:${IMAGE_TAG}"
SCANNER_DB_IMAGE="${REGISTRY}/scanner-db:${IMAGE_TAG}"
SCANNER_V4_IMAGE="${REGISTRY}/scanner-v4:${IMAGE_TAG}"
SCANNER_V4_DB_IMAGE="${REGISTRY}/scanner-v4-db:${IMAGE_TAG}"

echo "=== StackRox Dev Deploy ==="
echo "KUBECONFIG: ${KUBECONFIG}"
kubectl get nodes

# Clean up existing
kubectl delete namespace stackrox --wait=false 2>/dev/null || true
sleep 5

# Generate bundle
BUNDLE_DIR=$(mktemp -d)
echo "Generating bundle in ${BUNDLE_DIR}..."

"${ROXCTL}" central generate k8s pvc \
  --output-dir "${BUNDLE_DIR}" \
  --main-image "${MAIN_IMAGE}" \
  --central-db-image "${CENTRAL_DB_IMAGE}" \
  --scanner-image "${SCANNER_IMAGE}" \
  --scanner-db-image "${SCANNER_DB_IMAGE}" \
  --scanner-v4-image "${SCANNER_V4_IMAGE}" \
  --scanner-v4-db-image "${SCANNER_V4_DB_IMAGE}" \
  --enable-telemetry=false

# Run setup scripts
"${BUNDLE_DIR}/central/scripts/setup.sh" 2>/dev/null || true
"${BUNDLE_DIR}/scanner/scripts/setup.sh" 2>/dev/null || true

# Deploy
kubectl create -R -f "${BUNDLE_DIR}/central/" 2>/dev/null || true
kubectl create -R -f "${BUNDLE_DIR}/scanner/" 2>/dev/null || true

echo "Waiting for initial deployment..."
sleep 10

# Apply dev-friendly resource limits
echo "Applying dev resource limits..."
kubectl patch deploy central -n stackrox --type=json -p='[
  {"op": "replace", "path": "/spec/template/spec/containers/0/resources/requests/cpu", "value": "500m"},
  {"op": "replace", "path": "/spec/template/spec/containers/0/resources/requests/memory", "value": "1Gi"}
]'

kubectl patch deploy central-db -n stackrox --type=json -p='[
  {"op": "replace", "path": "/spec/template/spec/containers/0/resources/requests/cpu", "value": "500m"},
  {"op": "replace", "path": "/spec/template/spec/containers/0/resources/requests/memory", "value": "2Gi"}
]'

kubectl patch deploy scanner -n stackrox --type=json -p='[
  {"op": "replace", "path": "/spec/template/spec/containers/0/resources/requests/cpu", "value": "250m"},
  {"op": "replace", "path": "/spec/template/spec/containers/0/resources/requests/memory", "value": "500Mi"}
]'

kubectl patch deploy scanner-db -n stackrox --type=json -p='[
  {"op": "replace", "path": "/spec/template/spec/containers/0/resources/requests/cpu", "value": "100m"},
  {"op": "replace", "path": "/spec/template/spec/containers/0/resources/requests/memory", "value": "256Mi"}
]'

# Scale scanner to 1 replica and delete HPA
kubectl delete hpa scanner -n stackrox 2>/dev/null || true
kubectl scale deploy scanner -n stackrox --replicas=1

echo "Waiting for central..."
kubectl wait --for=condition=Available --timeout=300s deploy/central -n stackrox

PASSWORD=$(cat "${BUNDLE_DIR}/password")

echo ""
echo "=== StackRox Deployed ==="
kubectl get pods -n stackrox
echo ""
echo "Admin password: ${PASSWORD}"
echo ""
echo "Access:"
echo "  kubectl port-forward -n stackrox svc/central 8443:443"
echo "  https://localhost:8443"
echo "  admin / ${PASSWORD}"
