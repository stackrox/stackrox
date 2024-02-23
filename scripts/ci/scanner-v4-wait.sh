#!/usr/bin/env bash
set -eu

# Wait for Scanner V4 services to be up
#
# Usage:
#   scanner-v4-wait.sh [ <namespace> ]
#

SCRIPTS_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
# shellcheck source=../../scripts/ci/lib.sh
source "$SCRIPTS_ROOT/scripts/ci/lib.sh"

# wait_for_replicas(namespace, replication_method, workload_name, timeout)
#######################################
# Wait for Kubernetes workload replication. Logs info about replicas and readyReplicas. Call the exit
# Arguments:
#   namespace: Kubernetes namespace to monitor
#   replication_method: Name of the Kubernetes Kind that manages the workload, e.g., deployment, daemonset, etc.
#   workload_name: Name of the workload to monitor. Example: `deployment/central`, `central` would be the workload name
#   timeout: Monitoring timeout in seconds
# Returns:
#   0 if the workload is ready before the timeout
#   1 if the workload isn't ready before the timeout
#######################################
wait_for_replicas() {
    local namespace=$1
    local replication_method=$2
    local workload_name=$3
    local timeout=$4

    local start_time
    start_time="$(date '+%s')"

    while true; do
        local workload_json
        local expected_replicas
        local ready_replicas

        workload_json="$(kubectl -n "${namespace}" get "${replication_method}/${workload_name}" -o json)"
        expected_replicas="$(jq '.status.replicas' <<<"${workload_json}")"
        ready_replicas="$(jq '.status.readyReplicas' <<<"${workload_json}")"

        if [[ "${expected_replicas}" == "${ready_replicas}" ]]; then
            break
        fi

        info "${replication_method}/${workload_name} replicas: ${expected_replicas}"
        info "${replication_method}/${workload_name} readyReplicas: ${ready_replicas}"

        if (($(date '+%s') - start_time > timeout)); then
            return 1
        fi
    done
}

scanner_v4_wait() {
    local scanner_v4_namespace=${1:-stackrox}

    local scanner_v4_deployment_names=("scanner-v4-db" "scanner-v4-indexer" "scanner-v4-matcher")

    info "Waiting for Scanner V4 services in ${scanner_v4_namespace}"

    local deployment_name
    local timeout=1200
    for deployment_name in "${scanner_v4_deployment_names[@]}"; do
        info "Waiting for ${deployment_name} in ${scanner_v4_namespace}"

        if ! wait_for_replicas "${scanner_v4_namespace}" "deployment" "${deployment_name}" "${timeout}"; then
            info "Timeout reached. Printing debug info"

            kubectl -n "${scanner_v4_namespace}" get pod -o wide
            kubectl -n "${scanner_v4_namespace}" get deployment -o wide

            die "Timed out after $((timeout / 60))m"
        fi

        info "${deployment_name} is running"
    done

    info "All Scanner V4 services are running"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    scanner_v4_wait "$@"
fi
