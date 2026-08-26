#!/usr/bin/env bash
set -euo pipefail

# Cleanup berserker DaemonSet and ConfigMap resources
#
# Usage: cleanup-berserker.sh [workload_name] [namespace]
#   workload_name: e.g., "file-activity-100" (optional - if omitted, cleanup all)
#   namespace: default is "stackrox"
#
# Examples:
#   ./cleanup-berserker.sh file-activity-100 stackrox  # Cleanup specific workload
#   ./cleanup-berserker.sh "" stackrox                  # Cleanup all berserker resources

WORKLOAD_NAME="${1:-}"
NAMESPACE="${2:-stackrox}"

echo "================================================================"
if [[ -z "${WORKLOAD_NAME}" ]]; then
    echo "Cleaning up ALL berserker resources in namespace: ${NAMESPACE}"
else
    echo "Cleaning up berserker workload: ${WORKLOAD_NAME}"
    echo "Namespace: ${NAMESPACE}"
fi
echo "================================================================"

cleanup_workload() {
    local workload="$1"
    local ns="$2"

    echo "Deleting DaemonSet: berserker-file-activity-${workload}"
    kubectl delete daemonset "berserker-file-activity-${workload}" -n "${ns}" --ignore-not-found=true

    echo "Deleting ConfigMap: berserker-file-activity-config-${workload}"
    kubectl delete configmap "berserker-file-activity-config-${workload}" -n "${ns}" --ignore-not-found=true

    echo "Waiting for pods to terminate..."
    local max_wait=60
    local elapsed=0
    while kubectl get pods -n "${ns}" -l "app=berserker-file-activity-${workload}" 2>/dev/null | grep -q berserker; do
        if [[ ${elapsed} -ge ${max_wait} ]]; then
            echo "Warning: Pods still running after ${max_wait} seconds"
            kubectl get pods -n "${ns}" -l "app=berserker-file-activity-${workload}" || true
            break
        fi
        echo "  Waiting for pods to terminate (${elapsed}s / ${max_wait}s)..."
        sleep 2
        ((elapsed+=2))
    done

    echo "Cleanup complete for workload: ${workload}"
}

if [[ -z "${WORKLOAD_NAME}" ]]; then
    # Cleanup all berserker resources
    echo "Finding all berserker DaemonSets..."
    DAEMONSETS=$(kubectl get daemonsets -n "${NAMESPACE}" -o name | grep "berserker-file-activity" || true)

    if [[ -z "${DAEMONSETS}" ]]; then
        echo "No berserker DaemonSets found in namespace: ${NAMESPACE}"
    else
        for ds in ${DAEMONSETS}; do
            # Extract workload name from DaemonSet name
            # daemonset.apps/berserker-file-activity-file-activity-100 -> file-activity-100
            workload=$(echo "${ds}" | sed 's|daemonset.apps/berserker-file-activity-||')
            cleanup_workload "${workload}" "${NAMESPACE}"
        done
    fi

    # Cleanup any orphaned ConfigMaps
    echo "Finding orphaned berserker ConfigMaps..."
    CONFIGMAPS=$(kubectl get configmaps -n "${NAMESPACE}" -o name | grep "berserker-file-activity-config" || true)
    if [[ -n "${CONFIGMAPS}" ]]; then
        for cm in ${CONFIGMAPS}; do
            echo "Deleting orphaned ConfigMap: ${cm}"
            kubectl delete "${cm}" -n "${NAMESPACE}" --ignore-not-found=true
        done
    fi
else
    # Cleanup specific workload
    cleanup_workload "${WORKLOAD_NAME}" "${NAMESPACE}"
fi

echo "================================================================"
echo "Cleanup verification:"
echo "================================================================"
echo "Remaining berserker DaemonSets:"
kubectl get daemonsets -n "${NAMESPACE}" | grep berserker || echo "  None"
echo ""
echo "Remaining berserker ConfigMaps:"
kubectl get configmaps -n "${NAMESPACE}" | grep berserker || echo "  None"
echo ""
echo "Remaining berserker pods:"
kubectl get pods -n "${NAMESPACE}" -l "app" | grep berserker || echo "  None"
echo "================================================================"
echo "Cleanup complete!"
echo "================================================================"
