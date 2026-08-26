#!/usr/bin/env bash
set -euo pipefail

# Deploy berserker DaemonSet for file activity load testing
#
# Usage: deploy-berserker.sh <workload_name> [namespace]
#   workload_name: e.g., "file-activity-100"
#   namespace: default is "stackrox"
#
# Example: ./deploy-berserker.sh file-activity-100 stackrox

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${DIR}/../.." && pwd)"

WORKLOAD_NAME="${1:-}"
NAMESPACE="${2:-stackrox}"

if [[ -z "${WORKLOAD_NAME}" ]]; then
    echo "Error: workload_name is required"
    echo "Usage: $0 <workload_name> [namespace]"
    echo "Example: $0 file-activity-100 stackrox"
    exit 1
fi

WORKLOAD_FILE="${REPO_ROOT}/scale/berserker/workloads/${WORKLOAD_NAME}.ber"
CONFIGMAP_TEMPLATE="${REPO_ROOT}/scale/berserker/deployments/berserker-configmap-template.yaml"
DAEMONSET_TEMPLATE="${REPO_ROOT}/scale/berserker/deployments/berserker-daemonset-template.yaml"

echo "================================================================"
echo "Deploying berserker workload: ${WORKLOAD_NAME}"
echo "Namespace: ${NAMESPACE}"
echo "================================================================"

# Validate workload file exists
if [[ ! -f "${WORKLOAD_FILE}" ]]; then
    echo "Error: Workload file not found: ${WORKLOAD_FILE}"
    echo "Available workloads:"
    ls -1 "${REPO_ROOT}/scale/berserker/workloads/" || true
    exit 1
fi

# Validate templates exist
if [[ ! -f "${CONFIGMAP_TEMPLATE}" || ! -f "${DAEMONSET_TEMPLATE}" ]]; then
    echo "Error: Template files not found"
    echo "Expected: ${CONFIGMAP_TEMPLATE}"
    echo "Expected: ${DAEMONSET_TEMPLATE}"
    exit 1
fi

# Read workload file content
echo "Reading workload file: ${WORKLOAD_FILE}"
WORKLOAD_CONTENT=$(cat "${WORKLOAD_FILE}")

# Delete existing resources if they exist (idempotent)
echo "Cleaning up existing resources (if any)..."
kubectl delete daemonset "berserker-file-activity-${WORKLOAD_NAME}" -n "${NAMESPACE}" --ignore-not-found=true
kubectl delete configmap "berserker-file-activity-config-${WORKLOAD_NAME}" -n "${NAMESPACE}" --ignore-not-found=true

# Wait for pods to terminate
echo "Waiting for old pods to terminate..."
while kubectl get pods -n "${NAMESPACE}" -l "app=berserker-file-activity-${WORKLOAD_NAME}" 2>/dev/null | grep -q berserker; do
    echo "  Still waiting for pods to terminate..."
    sleep 2
done

# Create ConfigMap
echo "Creating ConfigMap..."
# We need to indent the workload content by 4 spaces for proper YAML formatting
INDENTED_WORKLOAD=$(echo "${WORKLOAD_CONTENT}" | sed 's/^/    /')
sed -e "s/WORKLOAD_NAME/${WORKLOAD_NAME}/g" \
    -e "s/NAMESPACE/${NAMESPACE}/g" \
    "${CONFIGMAP_TEMPLATE}" | \
    sed "/WORKLOAD_CONTENT/d" | \
    sed "/workload.ber: |/a\\
${INDENTED_WORKLOAD}" | \
    kubectl apply -f -

# Create DaemonSet
echo "Creating DaemonSet..."
sed -e "s/WORKLOAD_NAME/${WORKLOAD_NAME}/g" \
    -e "s/NAMESPACE/${NAMESPACE}/g" \
    "${DAEMONSET_TEMPLATE}" | \
    kubectl apply -f -

# Wait for DaemonSet rollout
echo "Waiting for DaemonSet rollout..."
if ! kubectl rollout status daemonset/berserker-file-activity-${WORKLOAD_NAME} -n "${NAMESPACE}" --timeout=120s; then
    echo "Error: DaemonSet rollout failed"
    echo "DaemonSet status:"
    kubectl get daemonset "berserker-file-activity-${WORKLOAD_NAME}" -n "${NAMESPACE}" || true
    echo "Pod status:"
    kubectl get pods -n "${NAMESPACE}" -l "app=berserker-file-activity-${WORKLOAD_NAME}" || true
    echo "Pod logs (if available):"
    kubectl logs -n "${NAMESPACE}" -l "app=berserker-file-activity-${WORKLOAD_NAME}" --tail=50 || true
    exit 1
fi

# Verify pod count
NODE_COUNT=$(kubectl get nodes --no-headers | wc -l)
POD_COUNT=$(kubectl get pods -n "${NAMESPACE}" -l "app=berserker-file-activity-${WORKLOAD_NAME}" --no-headers | wc -l)

echo "================================================================"
echo "Deployment successful!"
echo "  Nodes in cluster: ${NODE_COUNT}"
echo "  Berserker pods running: ${POD_COUNT}"
echo "  DaemonSet: berserker-file-activity-${WORKLOAD_NAME}"
echo "  ConfigMap: berserker-file-activity-config-${WORKLOAD_NAME}"
echo "================================================================"

if [[ "${POD_COUNT}" -ne "${NODE_COUNT}" ]]; then
    echo "WARNING: Pod count (${POD_COUNT}) does not match node count (${NODE_COUNT})"
    echo "This may affect the expected event rate."
fi

echo "Berserker pods:"
kubectl get pods -n "${NAMESPACE}" -l "app=berserker-file-activity-${WORKLOAD_NAME}" -o wide
