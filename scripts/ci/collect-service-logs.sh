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
# $ ./scripts/ci/collect-service-logs.sh central deployment
#
# Assumptions:
# - Must be called from the root of the Apollo git repository.
# - Logs are saved under /tmp/k8s-service-logs/

usage() {
    echo "./scripts/ci/collect-service-logs.sh <service name> <k8s type>"
    echo "e.g. ./scripts/ci/collect-service-logs.sh central deployment"
}

main() (
    if [ -z ${2+1} ]; then
        usage
        exit 1
    fi
    service_name="$1"
    type="$2"

    log_dir="/tmp/k8s-service-logs/${service_name}"
    mkdir -p "$log_dir"

	set +e
    kubectl describe "${type}/${service_name}" -n stackrox > "${log_dir}/describe.log"
    for pod in $(kubectl -n stackrox get po | grep "${service_name}" | awk '{print $1}'); do
        for ctr in $(kubectl -n stackrox get po $pod -o jsonpath='{.status.containerStatuses[*].name}'); do
            kubectl -n stackrox logs "po/${pod}" -c "$ctr" > "${log_dir}/${pod}-${ctr}.log"
            kubectl -n stackrox logs "po/${pod}" -p -c "$ctr" > "${log_dir}/${pod}-${ctr}-previous.log"
        done
    done
    find "${log_dir}" -type f -size 0 -delete
)

main "$@"
