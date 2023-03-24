#!/usr/bin/env bash
# shellcheck disable=SC1091
set -eu

TEST_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
source "$TEST_ROOT/scripts/lib.sh"
source "$TEST_ROOT/tests/e2e/separate-clusters.sh"

# Collect Service Logs From QA namespaces script
#
# Extracts service logs from QA namespaces and saves them off for
# future examination.
#
# Usage:
#   collect-qa-service-logs.sh [DIR]
#
# Example:
# $ ./scripts/ci/collect-qa-service-logs.sh
#
# Assumptions:
# - Must be called from the root of the Apollo git repository.
# - Logs are saved under /tmp/k8s-service-logs/ or DIR if passed

main() {
    if separate_clusters_test; then
        # QA test namespaces are created in the sensor cluster
        target_cluster "sensor"
        # Avoid checking both clusters in collect-service-logs.sh
        export TARGET_CLUSTER="sensor"
    fi
	set +e
    for ns in $(kubectl get ns -o json | jq -r '.items[].metadata.name' | grep -E '^qa'); do
        echo "Collecting from namespace: ${ns}"
        ./scripts/ci/collect-service-logs.sh "${ns}" $@
    done
}

main "$@"
