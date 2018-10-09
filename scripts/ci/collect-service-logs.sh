#!/bin/sh
set -eu

# Collect Service Logs script
#
# Extracts service logs from the given Kubernetes cluster and saves them off for
# future examination.
#
# Usage:
#   collect-service-logs.sh SERVICE
#
# Example:
# $ ./scripts/ci/collect-service-logs.sh central
#
# Assumptions:
# - Must be called from the root of the Apollo git repository.
# - Logs are saved under /tmp/k8s-service-logs/

main() {
    service_name="$1"
    if [ -z "$service_name" ]; then
        echo no service specified
        return 1
    fi

    log_dir="/tmp/k8s-service-logs/${service_name}"
    mkdir -p "$log_dir"

    kubectl describe "deploy/${service_name}" -n stackrox > "${log_dir}/describe.log"
    kubectl     logs "deploy/${service_name}" "${service_name}" -n stackrox > "${log_dir}/${service_name}.log"
    kubectl  logs -p "deploy/${service_name}" -n stackrox > "${log_dir}/logs_previous.log" 2>/dev/null || true
}

main "$@"
