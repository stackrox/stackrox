#!/bin/sh
set -eu

# Collect Service Logs From QA namespaces script
#
# Extracts service logs from QA namespaces and saves them off for
# future examination.
#
# Usage:
#   collect-qa-service-logs.sh DIR
#
# Example:
# $ ./scripts/ci/collect-qa-service-logs.sh /tmp/some-directory
#
# Assumptions:
# - Must be called from the root of the Apollo git repository.
# - Requires a single argument: path to save logs to.

main() {
	set +e
    for ns in $(kubectl get ns -o json | jq -r '.items[].metadata.name' | grep -E '^qa'); do
        echo "Collecting from namespace: ${ns}"
        ./scripts/ci/collect-service-logs.sh "${ns}" "$@"
    done
}

main "$@"
