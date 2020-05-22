#!/bin/sh
set -eu

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
	set +e
    for ns in $(kubectl get ns | tail +2 | egrep '^qa' | awk '{print $1}'); do
        echo "Collecting from namespace: ${ns}"
        ./scripts/ci/collect-service-logs.sh "${ns}" $@
    done
}

main "$@"
