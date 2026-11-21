#!/usr/bin/env bash
set -euo pipefail

# Deploy berserker DaemonSet for file activity load testing
#
# Usage: deploy-berserker.sh <workload_name> [namespace] [instance_name]
#   workload_name: e.g., "file-activity-100" (selects the .ber workload file)
#   namespace:     default is "stackrox"
#   instance_name: name for the k8s objects (DaemonSet/ConfigMap/label);
#                  defaults to workload_name. Use a distinct instance name to run
#                  several deployments off the same workload file, which gives each
#                  its own StackRox deployment ID (e.g. file-activity-100-d1,
#                  file-activity-100-d2) so alerts shard across separate workers.
#
# Example: ./deploy-berserker.sh file-activity-100 stackrox file-activity-100-d2

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${DIR}/../.." && pwd)"

WORKLOAD_NAME="${1:-}"
NAMESPACE="${2:-stackrox}"
# The .ber file is chosen by WORKLOAD_NAME; the k8s object names use INSTANCE_NAME.
# Keeping them separate lets multiple deployments share one workload definition.
INSTANCE_NAME="${3:-$WORKLOAD_NAME}"

if [[ -z "${WORKLOAD_NAME}" ]]; then
    echo "Error: workload_name is required"
    echo "Usage: $0 <workload_name> [namespace] [instance_name]"
    echo "Example: $0 file-activity-100 stackrox"
    exit 1
fi

WORKLOAD_FILE="${REPO_ROOT}/scale/berserker/workloads/${WORKLOAD_NAME}.ber"
DAEMONSET_TEMPLATE="${REPO_ROOT}/scale/berserker/deployments/berserker-daemonset-template.yaml"

echo "================================================================"
echo "Deploying berserker workload: ${WORKLOAD_NAME}"
echo "Instance (k8s object) name: ${INSTANCE_NAME}"
echo "Namespace: ${NAMESPACE}"
echo "================================================================"

# Validate workload file exists
if [[ ! -f "${WORKLOAD_FILE}" ]]; then
    echo "Error: Workload file not found: ${WORKLOAD_FILE}"
    echo "Available workloads:"
    ls -1 "${REPO_ROOT}/scale/berserker/workloads/" || true
    exit 1
fi

# Validate template exists
if [[ ! -f "${DAEMONSET_TEMPLATE}" ]]; then
    echo "Error: Template file not found"
    echo "Expected: ${DAEMONSET_TEMPLATE}"
    exit 1
fi

echo "Using workload file: ${WORKLOAD_FILE}"

# Delete existing resources if they exist (idempotent)
echo "Cleaning up existing resources (if any)..."
kubectl delete daemonset "berserker-file-activity-${INSTANCE_NAME}" -n "${NAMESPACE}" --ignore-not-found=true
kubectl delete configmap "berserker-file-activity-config-${INSTANCE_NAME}" -n "${NAMESPACE}" --ignore-not-found=true

# Wait for pods to terminate
echo "Waiting for old pods to terminate..."
while kubectl get pods -n "${NAMESPACE}" -l "app=berserker-file-activity-${INSTANCE_NAME}" 2>/dev/null | grep -q berserker; do
    echo "  Still waiting for pods to terminate..."
    sleep 2
done

# Create ConfigMap directly from the workload file. --from-file safely handles
# arbitrary multi-line workload content, avoiding fragile sed/YAML templating.
echo "Creating ConfigMap..."
kubectl create configmap "berserker-file-activity-config-${INSTANCE_NAME}" \
    -n "${NAMESPACE}" \
    --from-file=workload.ber="${WORKLOAD_FILE}" \
    --dry-run=client -o yaml | kubectl apply -f -

# Create DaemonSet. The template's WORKLOAD_NAME placeholder names the k8s objects
# (DaemonSet/label/ConfigMap ref), so it is substituted with INSTANCE_NAME.
echo "Creating DaemonSet..."
sed -e "s/WORKLOAD_NAME/${INSTANCE_NAME}/g" \
    -e "s/NAMESPACE/${NAMESPACE}/g" \
    "${DAEMONSET_TEMPLATE}" | \
    kubectl apply -f -

# Wait for DaemonSet rollout
echo "Waiting for DaemonSet rollout..."
if ! kubectl rollout status "daemonset/berserker-file-activity-${INSTANCE_NAME}" -n "${NAMESPACE}" --timeout=120s; then
    echo "Error: DaemonSet rollout failed"
    echo "DaemonSet status:"
    kubectl get daemonset "berserker-file-activity-${INSTANCE_NAME}" -n "${NAMESPACE}" || true
    echo "Pod status:"
    kubectl get pods -n "${NAMESPACE}" -l "app=berserker-file-activity-${INSTANCE_NAME}" || true
    echo "Pod logs (if available):"
    kubectl logs -n "${NAMESPACE}" -l "app=berserker-file-activity-${INSTANCE_NAME}" --tail=50 || true
    exit 1
fi

# Verify pod count
NODE_COUNT=$(kubectl get nodes --no-headers | wc -l)
POD_COUNT=$(kubectl get pods -n "${NAMESPACE}" -l "app=berserker-file-activity-${INSTANCE_NAME}" --no-headers | wc -l)

echo "================================================================"
echo "Deployment successful!"
echo "  Nodes in cluster: ${NODE_COUNT}"
echo "  Berserker pods running: ${POD_COUNT}"
echo "  DaemonSet: berserker-file-activity-${INSTANCE_NAME}"
echo "  ConfigMap: berserker-file-activity-config-${INSTANCE_NAME}"
echo "================================================================"

if [[ "${POD_COUNT}" -ne "${NODE_COUNT}" ]]; then
    echo "WARNING: Pod count (${POD_COUNT}) does not match node count (${NODE_COUNT})"
    echo "This may affect the expected event rate."
fi

echo "Berserker pods:"
kubectl get pods -n "${NAMESPACE}" -l "app=berserker-file-activity-${INSTANCE_NAME}" -o wide
