#!/bin/sh
set -eu

# Collect Service Logs script
#
# Extracts service logs from the given Kubernetes cluster and saves them off for
# future examination.
#
# Usage:
#   collect-service-logs.sh NAMESPACE
#
# Example:
# $ ./scripts/ci/collect-service-logs.sh stackrox
#
# Assumptions:
# - Must be called from the root of the Apollo git repository.
# - Logs are saved under /tmp/k8s-service-logs/

usage() {
    echo "./scripts/ci/collect-service-logs.sh <namespace>"
    echo "e.g. ./scripts/ci/collect-service-logs.sh stackrox"
}

main() {
    set -x
    namespace="$1"
    if [ -z "${namespace}" ]; then
        usage
        exit 1
    fi

    log_dir="/tmp/k8s-service-logs/${namespace}"
    mkdir -p "$log_dir"

	set +e
    for pod in $(kubectl -n "${namespace}" get po | tail +2 | awk '{print $1}'); do
        kubectl describe po "${pod}" -n "${namespace}" > "${log_dir}/${pod}_describe.log"
        for ctr in $(kubectl -n "${namespace}" get po $pod -o jsonpath='{.status.containerStatuses[*].name}'); do
            kubectl -n "${namespace}" logs "po/${pod}" -c "$ctr" > "${log_dir}/${pod}-${ctr}.log"
            kubectl -n "${namespace}" logs "po/${pod}" -p -c "$ctr" > "${log_dir}/${pod}-${ctr}-previous.log"
        done
    done
    find "${log_dir}" -type f -size 0 -delete
}

main "$@"
